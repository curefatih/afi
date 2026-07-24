package llm

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/aws/smithy-go"
	"github.com/curefatih/afi/internal/dataplane/ir"
	"github.com/curefatih/afi/internal/snapshot"
)

type stubBedrockAPI struct {
	converse       func(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error)
	converseStream func(ctx context.Context, params *bedrockruntime.ConverseStreamInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseStreamOutput, error)
}

func (s *stubBedrockAPI) Converse(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
	if s.converse == nil {
		return nil, stubAPIError{code: "InternalServerException", msg: "converse not stubbed"}
	}
	return s.converse(ctx, params, optFns...)
}

func (s *stubBedrockAPI) ConverseStream(ctx context.Context, params *bedrockruntime.ConverseStreamInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseStreamOutput, error) {
	if s.converseStream == nil {
		return nil, stubAPIError{code: "InternalServerException", msg: "stream not stubbed"}
	}
	return s.converseStream(ctx, params, optFns...)
}

type stubAPIError struct {
	code, msg string
}

func (e stubAPIError) Error() string                 { return e.code + ": " + e.msg }
func (e stubAPIError) ErrorCode() string             { return e.code }
func (e stubAPIError) ErrorMessage() string          { return e.msg }
func (e stubAPIError) ErrorFault() smithy.ErrorFault { return smithy.FaultUnknown }

func TestBedrockChatIRNonStream(t *testing.T) {
	api := &stubBedrockAPI{
		converse: func(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			if aws.ToString(params.ModelId) != "anthropic.claude-3-haiku" {
				t.Fatalf("model=%q", aws.ToString(params.ModelId))
			}
			return &bedrockruntime.ConverseOutput{
				StopReason: types.StopReasonEndTurn,
				Usage: &types.TokenUsage{
					InputTokens: aws.Int32(2), OutputTokens: aws.Int32(4), TotalTokens: aws.Int32(6),
				},
				Output: &types.ConverseOutputMemberMessage{
					Value: types.Message{
						Role: types.ConversationRoleAssistant,
						Content: []types.ContentBlock{
							&types.ContentBlockMemberText{Value: "hello from bedrock"},
						},
					},
				},
			}, nil
		},
	}
	client := NewBedrockClient(nil)
	client.API = api
	result, err := client.ChatIR(t.Context(), snapshot.Provider{
		ID: "b", Type: "bedrock", BaseURL: "https://bedrock-runtime.us-west-2.amazonaws.com",
	}, "anthropic.claude-3-haiku", ir.ChatRequest{
		Messages: []ir.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Response == nil || result.Response.Content != "hello from bedrock" {
		t.Fatalf("result=%+v", result)
	}
	if result.Response.Usage.PromptTokens != 2 || result.Response.Usage.CompletionTokens != 4 {
		t.Fatalf("usage=%+v", result.Response.Usage)
	}
}

func TestBedrockChatIRUpstreamError(t *testing.T) {
	api := &stubBedrockAPI{
		converse: func(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
			return nil, stubAPIError{code: "ValidationException", msg: "bad request"}
		},
	}
	client := NewBedrockClient(nil)
	client.API = api
	result, err := client.ChatIR(t.Context(), snapshot.Provider{
		ID: "b", Type: "bedrock", BaseURL: "https://bedrock-runtime.us-west-2.amazonaws.com",
	}, "m", ir.ChatRequest{
		Messages: []ir.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.StatusCode != 400 {
		t.Fatalf("status=%d body=%s", result.StatusCode, result.ErrorBody)
	}
	var body map[string]any
	if err := json.Unmarshal(result.ErrorBody, &body); err != nil {
		t.Fatal(err)
	}
	if body["message"] != "bad request" || body["code"] != "ValidationException" {
		t.Fatalf("body=%v", body)
	}
}

func TestBedrockChatIRStreamAPIError(t *testing.T) {
	api := &stubBedrockAPI{
		converseStream: func(ctx context.Context, params *bedrockruntime.ConverseStreamInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseStreamOutput, error) {
			return nil, stubAPIError{code: "ThrottlingException", msg: "slow down"}
		},
	}
	client := NewBedrockClient(nil)
	client.API = api
	result, err := client.ChatIR(t.Context(), snapshot.Provider{
		ID: "b", Type: "bedrock", BaseURL: "https://bedrock-runtime.us-west-2.amazonaws.com",
	}, "m", ir.ChatRequest{
		Stream:   true,
		Messages: []ir.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.StatusCode != 429 {
		t.Fatalf("status=%d", result.StatusCode)
	}
}

func TestResolveAWSConfigStaticCredentials(t *testing.T) {
	client := NewBedrockClient(nil)
	cfg, err := client.resolveAWSConfig(t.Context(), snapshot.Provider{
		ID:           "bedrock",
		BaseURL:      "https://bedrock-runtime.eu-west-1.amazonaws.com",
		InlineAPIKey: "AKIA_TEST:secret-test:session-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Region != "eu-west-1" {
		t.Fatalf("region=%q", cfg.Region)
	}
	creds, err := cfg.Credentials.Retrieve(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if creds.AccessKeyID != "AKIA_TEST" || creds.SecretAccessKey != "secret-test" ||
		creds.SessionToken != "session-test" {
		t.Fatalf("credentials=%+v", creds)
	}
}

func TestResolveAWSConfigDefaultCredentialChain(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIA_ENV")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret-env")
	t.Setenv("AWS_SESSION_TOKEN", "")

	client := NewBedrockClient(nil)
	cfg, err := client.resolveAWSConfig(t.Context(), snapshot.Provider{
		ID:      "bedrock",
		BaseURL: "https://bedrock-runtime.us-east-2.amazonaws.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	creds, err := cfg.Credentials.Retrieve(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Region != "us-east-2" || creds.AccessKeyID != "AKIA_ENV" ||
		creds.SecretAccessKey != "secret-env" {
		t.Fatalf("region=%q credentials=%+v", cfg.Region, creds)
	}
}

func TestResolveAWSConfigRejectsMalformedStaticCredentials(t *testing.T) {
	client := NewBedrockClient(nil)
	_, err := client.resolveAWSConfig(t.Context(), snapshot.Provider{
		ID:           "bedrock",
		BaseURL:      "https://bedrock-runtime.us-east-1.amazonaws.com",
		InlineAPIKey: "not-a-static-credential",
	})
	if err == nil {
		t.Fatal("expected malformed credentials error")
	}
}
