package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/smithy-go"
	"github.com/curefatih/afi/internal/adapters/secrets"
	"github.com/curefatih/afi/internal/dataplane/ir"
	"github.com/curefatih/afi/internal/snapshot"
)

// BedrockConverseAPI is the subset of Bedrock Runtime used by BedrockClient.
type BedrockConverseAPI interface {
	Converse(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error)
	ConverseStream(ctx context.Context, params *bedrockruntime.ConverseStreamInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseStreamOutput, error)
}

// BedrockClient calls Amazon Bedrock Converse / ConverseStream.
type BedrockClient struct {
	Secrets secrets.Resolver
	// API is optional; when nil, a client is built per request from provider config.
	API BedrockConverseAPI
	// NewAPI builds a Bedrock Runtime client; tests may override.
	NewAPI func(ctx context.Context, provider snapshot.Provider) (BedrockConverseAPI, error)
}

// NewBedrockClient constructs a Bedrock adapter with a shared secret resolver.
func NewBedrockClient(sec secrets.Resolver) *BedrockClient {
	if sec == nil {
		sec = secrets.Default()
	}
	c := &BedrockClient{Secrets: sec}
	c.NewAPI = c.defaultNewAPI
	return c
}

func (c *BedrockClient) defaultNewAPI(ctx context.Context, provider snapshot.Provider) (BedrockConverseAPI, error) {
	cfg, err := c.resolveAWSConfig(ctx, provider)
	if err != nil {
		return nil, err
	}
	return bedrockruntime.NewFromConfig(cfg), nil
}

func (c *BedrockClient) apiFor(ctx context.Context, provider snapshot.Provider) (BedrockConverseAPI, error) {
	if c.API != nil {
		return c.API, nil
	}
	if c.NewAPI == nil {
		c.NewAPI = c.defaultNewAPI
	}
	return c.NewAPI(ctx, provider)
}

// ChatIR runs chat against Bedrock Converse using IR.
func (c *BedrockClient) ChatIR(ctx context.Context, provider snapshot.Provider, targetModel string, req ir.ChatRequest) (ir.ChatResult, error) {
	api, err := c.apiFor(ctx, provider)
	if err != nil {
		return ir.ChatResult{}, err
	}
	if req.Stream {
		return c.chatIRStream(ctx, api, targetModel, req)
	}
	in, err := irToConverseInput(targetModel, req)
	if err != nil {
		return ir.ChatResult{}, err
	}
	out, err := api.Converse(ctx, in)
	if err != nil {
		return bedrockAPIErrorResult(err)
	}
	mapped, err := converseOutputToIR(out, targetModel)
	if err != nil {
		return ir.ChatResult{}, err
	}
	return ir.ChatResult{StatusCode: http.StatusOK, Response: &mapped}, nil
}

func (c *BedrockClient) chatIRStream(ctx context.Context, api BedrockConverseAPI, targetModel string, req ir.ChatRequest) (ir.ChatResult, error) {
	in, err := irToConverseStreamInput(targetModel, req)
	if err != nil {
		return ir.ChatResult{}, err
	}
	out, err := api.ConverseStream(ctx, in)
	if err != nil {
		return bedrockAPIErrorResult(err)
	}
	ch := make(chan ir.StreamEvent, 16)
	go func() {
		defer close(ch)
		defer func() { _ = out.GetStream().Close() }()
		id := "chatcmpl-bedrock"
		nextToolIndex := 0
		toolIndexes := map[int]int{}
		started := false
		for ev := range out.GetStream().Events() {
			events := mapConverseStreamEvent(ev, targetModel, id, &nextToolIndex, toolIndexes)
			for _, e := range events {
				if e.Kind == ir.StreamMessageStart {
					started = true
				}
				if !started && e.Kind != ir.StreamError {
					started = true
					ch <- ir.StreamEvent{Kind: ir.StreamMessageStart, ID: id, Model: targetModel, Role: "assistant"}
				}
				ch <- e
			}
		}
		if err := out.GetStream().Err(); err != nil {
			ch <- ir.StreamEvent{Kind: ir.StreamError, Err: err}
		}
	}()
	return ir.ChatResult{StatusCode: http.StatusOK, Events: ch}, nil
}

func bedrockAPIErrorResult(err error) (ir.ChatResult, error) {
	status := http.StatusBadGateway
	msg := err.Error()
	code := "api_error"
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code = apiErr.ErrorCode()
		msg = apiErr.ErrorMessage()
		if msg == "" {
			msg = apiErr.Error()
		}
		status = bedrockErrorHTTPStatus(code)
	}
	body, _ := json.Marshal(map[string]any{
		"message": msg,
		"code":    code,
		"type":    code,
	})
	return ir.ChatResult{StatusCode: status, ErrorBody: body}, nil
}

func bedrockErrorHTTPStatus(code string) int {
	switch strings.ToLower(code) {
	case "accessdeniedexception", "unauthorizedexception", "unrecognizedclientexception":
		return http.StatusUnauthorized
	case "validationexception", "invalidparameterexception", "modelerrorexception", "modelnotreadyexception":
		return http.StatusBadRequest
	case "resourcenotfoundexception", "resourcenotfound":
		return http.StatusNotFound
	case "throttlingexception", "toomanyrequestsexception", "servicequotaexceededexception":
		return http.StatusTooManyRequests
	case "modeltimeoutexception", "serviceunavailableexception", "internalserverexception":
		return http.StatusBadGateway
	default:
		return http.StatusBadGateway
	}
}

func (c *BedrockClient) resolveAWSConfig(ctx context.Context, provider snapshot.Provider) (aws.Config, error) {
	region := regionFromBaseURL(provider.BaseURL)
	secret, err := c.resolveSecret(ctx, provider)
	if err != nil {
		return aws.Config{}, err
	}

	var opts []func(*config.LoadOptions) error
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}
	if secret != "" {
		accessKey, secretKey, sessionToken, ok := parseAWSStaticCreds(secret)
		if !ok {
			return aws.Config{}, fmt.Errorf("bedrock: static credentials must be accessKeyID:secretAccessKey[:sessionToken]")
		}
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, sessionToken),
		))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("aws config: %w", err)
	}
	if cfg.Region == "" && region != "" {
		cfg.Region = region
	}
	if cfg.Region == "" {
		return aws.Config{}, fmt.Errorf("bedrock: region required (set provider base_url to https://bedrock-runtime.{region}.amazonaws.com or AWS_REGION)")
	}
	return cfg, nil
}

func (c *BedrockClient) resolveSecret(ctx context.Context, provider snapshot.Provider) (string, error) {
	if provider.InlineAPIKey != "" {
		return provider.InlineAPIKey, nil
	}
	ref := strings.TrimSpace(provider.APIKeyEnv)
	if ref == "" {
		return "", nil
	}
	if c.Secrets == nil {
		return "", fmt.Errorf("secrets resolver not configured for provider %s", provider.ID)
	}
	return c.Secrets.Get(ctx, ref)
}

// parseAWSStaticCreds parses accessKeyID:secretAccessKey[:sessionToken].
// Returns ok=false when the string is empty or malformed (caller uses default chain).
func parseAWSStaticCreds(s string) (accessKey, secretKey, sessionToken string, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", "", "", false
	}
	parts := strings.SplitN(s, ":", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", false
	}
	if len(parts) == 3 {
		return parts[0], parts[1], parts[2], true
	}
	return parts[0], parts[1], "", true
}

// regionFromBaseURL extracts region from https://bedrock-runtime.{region}.amazonaws.com.
func regionFromBaseURL(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return ""
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	host := u.Hostname()
	const prefix = "bedrock-runtime."
	const suffix = ".amazonaws.com"
	if strings.HasPrefix(host, prefix) && strings.HasSuffix(host, suffix) {
		return strings.TrimSuffix(strings.TrimPrefix(host, prefix), suffix)
	}
	return ""
}
