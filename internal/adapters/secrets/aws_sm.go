package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// SecretsManagerAPI is the subset of AWS Secrets Manager used by AWSSM.
type SecretsManagerAPI interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

// AWSSM resolves aws-sm://{region}/{secret-id}[#jsonKey] via AWS Secrets Manager.
type AWSSM struct {
	Client SecretsManagerAPI
	Region string // default region when ref omits one (unusual)
}

// NewAWSSMFromEnv loads the default AWS config when enabled is true.
func NewAWSSMFromEnv(ctx context.Context, enabled bool, region string) (*AWSSM, error) {
	if !enabled {
		return nil, nil
	}
	opts := []func(*config.LoadOptions) error{}
	if region = strings.TrimSpace(region); region != "" {
		opts = append(opts, config.WithRegion(region))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("aws config: %w", err)
	}
	return &AWSSM{
		Client: secretsmanager.NewFromConfig(cfg),
		Region: cfg.Region,
	}, nil
}

func (a *AWSSM) Get(ctx context.Context, ref string) (string, error) {
	if a == nil || a.Client == nil {
		return "", fmt.Errorf("aws-sm secret resolver not configured")
	}
	p, err := ParseRef(ref)
	if err != nil {
		return "", err
	}
	if p.Scheme != "aws-sm" {
		return "", fmt.Errorf("expected aws-sm:// ref, got %q", ref)
	}
	// path: region/secret-id or secret-id (use default region)
	region := a.Region
	secretID := p.Path
	if i := strings.Index(p.Path, "/"); i > 0 {
		maybeRegion := p.Path[:i]
		rest := p.Path[i+1:]
		// Heuristic: AWS regions look like us-east-1, eu-west-1, ap-southeast-2
		if looksLikeAWSRegion(maybeRegion) && rest != "" {
			region = maybeRegion
			secretID = rest
		}
	}
	if secretID == "" {
		return "", fmt.Errorf("aws-sm ref missing secret id")
	}
	out, err := a.Client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretID),
	}, func(o *secretsmanager.Options) {
		if region != "" {
			o.Region = region
		}
	})
	if err != nil {
		return "", fmt.Errorf("aws secrets manager: %w", err)
	}
	var raw string
	if out.SecretString != nil {
		raw = *out.SecretString
	} else if len(out.SecretBinary) > 0 {
		raw = string(out.SecretBinary)
	}
	if raw == "" {
		return "", fmt.Errorf("aws secret %q is empty", secretID)
	}
	if p.Key == "" {
		return raw, nil
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return "", fmt.Errorf("aws secret is not JSON object for key %q: %w", p.Key, err)
	}
	v, ok := obj[p.Key]
	if !ok {
		return "", fmt.Errorf("aws secret missing key %q", p.Key)
	}
	return stringifySecret(v)
}

func looksLikeAWSRegion(s string) bool {
	parts := strings.Split(s, "-")
	if len(parts) < 3 {
		return false
	}
	last := parts[len(parts)-1]
	if len(last) == 0 {
		return false
	}
	for _, c := range last {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
