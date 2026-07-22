package secrets

import (
	"fmt"
	"net/url"
	"strings"
)

// ParsedRef is a scheme-routed secret reference.
type ParsedRef struct {
	Scheme string // env | aws-sm | hashicorp | "" (bare env name)
	Path   string // secret id / vault path (no leading slash required)
	Key    string // optional JSON object key after #
	Raw    string
}

// ParseRef parses secret references:
//
//	OPENAI_API_KEY
//	env://OPENAI_API_KEY
//	aws-sm://us-east-1/my/secret#api_key
//	hashicorp://secret/data/afi/openai#api_key
func ParseRef(ref string) (ParsedRef, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ParsedRef{}, fmt.Errorf("empty secret ref")
	}
	out := ParsedRef{Raw: ref}
	if !strings.Contains(ref, "://") {
		out.Scheme = "env"
		out.Path = ref
		return out, nil
	}
	u, err := url.Parse(ref)
	if err != nil {
		return ParsedRef{}, fmt.Errorf("invalid secret ref: %w", err)
	}
	out.Scheme = strings.ToLower(u.Scheme)
	hostPath := u.Host
	if u.Path != "" && u.Path != "/" {
		if hostPath != "" {
			hostPath = hostPath + u.Path
		} else {
			hostPath = strings.TrimPrefix(u.Path, "/")
		}
	}
	frag := u.Fragment
	if frag == "" && strings.Contains(hostPath, "#") {
		parts := strings.SplitN(hostPath, "#", 2)
		hostPath = parts[0]
		frag = parts[1]
	}
	out.Path = strings.TrimPrefix(hostPath, "/")
	out.Key = frag
	if out.Path == "" {
		return ParsedRef{}, fmt.Errorf("secret ref missing path")
	}
	switch out.Scheme {
	case "env", "aws-sm", "hashicorp":
	default:
		return ParsedRef{}, fmt.Errorf("unsupported secret ref scheme %q", out.Scheme)
	}
	return out, nil
}
