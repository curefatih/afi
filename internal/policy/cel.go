package policy

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/curefatih/afi/internal/snapshot"
	"github.com/google/cel-go/cel"
)

// Request is the CEL evaluation context for a gateway call.
type Request struct {
	Model   string
	Path    string
	Stream  bool
	Tags    map[string]string
	Headers map[string]string // lowercased inbound HTTP headers (sanitized)
}

// Credential is optional BYOK metadata for CEL (resolved before Open).
type Credential struct {
	ID           string
	Name         string
	StorageKind  string
	IsBYOK       bool
	ProviderType string
}

// Decision is the result of applying when/then policies in priority order.
type Decision struct {
	Allowed          bool
	DeniedBy         string
	CredentialName   string
	RequestHeaders   map[string]string // outbound headers to merge onto provider request
	MatchedAllowName string            // short-circuit allow policy name, if any
}

// Evaluator compiles and runs when/then CEL policies.
type Evaluator struct {
	env         *cel.Env
	boolCache   sync.Map // expression -> cel.Program (bool)
	stringCache sync.Map // expression -> cel.Program (string)
}

// NewEvaluator builds a shared CEL environment.
func NewEvaluator() (*Evaluator, error) {
	env, err := cel.NewEnv(
		cel.Variable("request", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("key", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("credential", cel.MapType(cel.StringType, cel.DynType)),
	)
	if err != nil {
		return nil, err
	}
	return &Evaluator{env: env}, nil
}

// Validate checks that expression is a boolean CEL program.
func Validate(expression string) error {
	ev, err := NewEvaluator()
	if err != nil {
		return err
	}
	_, err = ev.boolProgram(expression)
	return err
}

// ValidateString checks that expression is a string CEL program (for dynamic action values).
func ValidateString(expression string) error {
	ev, err := NewEvaluator()
	if err != nil {
		return err
	}
	_, err = ev.stringProgram(expression)
	return err
}

func (e *Evaluator) boolProgram(expression string) (cel.Program, error) {
	if expression == "" {
		return nil, fmt.Errorf("empty expression")
	}
	if v, ok := e.boolCache.Load(expression); ok {
		return v.(cel.Program), nil
	}
	ast, iss := e.env.Compile(expression)
	if iss != nil && iss.Err() != nil {
		return nil, iss.Err()
	}
	if ast.OutputType() != cel.BoolType {
		return nil, fmt.Errorf("expression must evaluate to bool, got %s", ast.OutputType())
	}
	prg, err := e.env.Program(ast)
	if err != nil {
		return nil, err
	}
	e.boolCache.Store(expression, prg)
	return prg, nil
}

func (e *Evaluator) stringProgram(expression string) (cel.Program, error) {
	if expression == "" {
		return nil, fmt.Errorf("empty expression")
	}
	if v, ok := e.stringCache.Load(expression); ok {
		return v.(cel.Program), nil
	}
	ast, iss := e.env.Compile(expression)
	if iss != nil && iss.Err() != nil {
		return nil, iss.Err()
	}
	// Map/index lookups are dyn; allow string or dyn and coerce at eval.
	outType := ast.OutputType()
	if outType != cel.StringType && outType != cel.DynType {
		return nil, fmt.Errorf("expression must evaluate to string, got %s", outType)
	}
	prg, err := e.env.Program(ast)
	if err != nil {
		return nil, err
	}
	e.stringCache.Store(expression, prg)
	return prg, nil
}

func sortedOrgPolicies(policies []snapshot.Policy, orgID string) []snapshot.Policy {
	var matched []snapshot.Policy
	for _, p := range policies {
		if !p.Enabled || p.OrganizationID != orgID {
			continue
		}
		matched = append(matched, p)
	}
	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].Priority != matched[j].Priority {
			return matched[i].Priority > matched[j].Priority
		}
		return matched[i].Name < matched[j].Name
	})
	return matched
}

func evalVars(key snapshot.APIKey, req Request, cred Credential) map[string]any {
	tags := map[string]any{}
	for k, v := range req.Tags {
		tags[k] = v
	}
	headers := map[string]any{}
	for k, v := range req.Headers {
		headers[k] = v
	}
	return map[string]any{
		"request": map[string]any{
			"model":   req.Model,
			"path":    req.Path,
			"stream":  req.Stream,
			"tags":    tags,
			"headers": headers,
		},
		"key": map[string]any{
			"id":              key.ID,
			"organization_id": key.OrganizationID,
			"project_id":      key.ProjectID,
			"kind":            key.Kind,
			"owner_user_id":   key.OwnerUserID,
			"name":            key.Name,
		},
		"credential": map[string]any{
			"id":            cred.ID,
			"name":          cred.Name,
			"storage_kind":  cred.StorageKind,
			"is_byok":       cred.IsBYOK,
			"provider_type": cred.ProviderType,
		},
	}
}

func (e *Evaluator) evalBool(expression string, vars map[string]any) (bool, error) {
	prg, err := e.boolProgram(expression)
	if err != nil {
		return false, err
	}
	out, _, err := prg.Eval(vars)
	if err != nil {
		return false, err
	}
	ok, valid := out.Value().(bool)
	if !valid {
		return false, fmt.Errorf("non-bool result")
	}
	return ok, nil
}

func (e *Evaluator) evalString(expression string, vars map[string]any) (string, error) {
	prg, err := e.stringProgram(expression)
	if err != nil {
		return "", err
	}
	out, _, err := prg.Eval(vars)
	if err != nil {
		return "", err
	}
	switch v := out.Value().(type) {
	case string:
		return v, nil
	case nil:
		return "", nil
	default:
		return fmt.Sprint(v), nil
	}
}

// resolveDynamicString returns static if set, otherwise evaluates expr as CEL string.
func (e *Evaluator) resolveDynamicString(static, expr string, vars map[string]any) (string, error) {
	expr = strings.TrimSpace(expr)
	if expr != "" {
		return e.evalString(expr, vars)
	}
	return static, nil
}

// Apply walks enabled policies by priority (desc). When Expression is true, runs
// each Then action in order:
//   - deny → stop, deny
//   - allow → stop, allow (short-circuit)
//   - set_header → set outbound header (later writes overwrite), continue
//   - use_credential → set credential name (later writes overwrite), continue
// Default (no deny / no short-circuit allow): allow.
func (e *Evaluator) Apply(policies []snapshot.Policy, key snapshot.APIKey, req Request, cred Credential) (Decision, error) {
	out := Decision{Allowed: true, RequestHeaders: map[string]string{}}
	vars := evalVars(key, req, cred)
	for _, p := range sortedOrgPolicies(policies, key.OrganizationID) {
		ok, err := e.evalBool(p.Expression, vars)
		if err != nil {
			return Decision{}, fmt.Errorf("policy %q: %w", p.Name, err)
		}
		if !ok {
			continue
		}
		actions := p.EffectiveActions()
		if len(actions) == 0 {
			return Decision{}, fmt.Errorf("policy %q: no then actions", p.Name)
		}
		for ai, act := range actions {
			stop, err := e.applyAction(p.Name, ai, act, vars, &out)
			if err != nil {
				return Decision{}, err
			}
			if stop {
				return out, nil
			}
		}
	}
	return out, nil
}

// applyAction runs one Then step. stop=true means the overall Apply should return.
func (e *Evaluator) applyAction(policyName string, index int, act snapshot.PolicyAction, vars map[string]any, out *Decision) (stop bool, err error) {
	action := strings.TrimSpace(strings.ToLower(act.Type))
	if action == "" {
		action = snapshot.PolicyActionDeny
	}
	label := fmt.Sprintf("%s.actions[%d]", policyName, index)
	switch action {
	case snapshot.PolicyActionDeny:
		*out = Decision{Allowed: false, DeniedBy: policyName}
		return true, nil
	case snapshot.PolicyActionAllow:
		out.MatchedAllowName = policyName
		return true, nil
	case snapshot.PolicyActionSetHeader:
		var cfg struct {
			Header    string `json:"header"`
			Value     string `json:"value"`
			ValueExpr string `json:"value_expr"`
		}
		if len(act.Config) > 0 {
			if err := json.Unmarshal(act.Config, &cfg); err != nil {
				return false, fmt.Errorf("policy %q: invalid set_header config: %w", label, err)
			}
		}
		h := strings.TrimSpace(cfg.Header)
		if h == "" {
			return false, fmt.Errorf("policy %q: set_header missing header", label)
		}
		val, err := e.resolveDynamicString(cfg.Value, cfg.ValueExpr, vars)
		if err != nil {
			return false, fmt.Errorf("policy %q: set_header value: %w", label, err)
		}
		out.RequestHeaders[h] = val
		return false, nil
	case snapshot.PolicyActionUseCredential:
		var cfg struct {
			CredentialName     string `json:"credential_name"`
			CredentialNameExpr string `json:"credential_name_expr"`
		}
		if len(act.Config) > 0 {
			if err := json.Unmarshal(act.Config, &cfg); err != nil {
				return false, fmt.Errorf("policy %q: invalid use_credential config: %w", label, err)
			}
		}
		name, err := e.resolveDynamicString(cfg.CredentialName, cfg.CredentialNameExpr, vars)
		if err != nil {
			return false, fmt.Errorf("policy %q: use_credential name: %w", label, err)
		}
		name = strings.TrimSpace(name)
		if name == "" {
			return false, fmt.Errorf("policy %q: use_credential resolved empty credential name", label)
		}
		out.CredentialName = name
		return false, nil
	default:
		return false, fmt.Errorf("policy %q: unknown action %q", label, act.Type)
	}
}

// CredentialFromSnapshot builds CEL credential metadata without opening the secret.
func CredentialFromSnapshot(snap *snapshot.Snapshot, providerType string, key snapshot.APIKey, overrideName string) Credential {
	out := Credential{ProviderType: providerType}
	if snap == nil || providerType == "" {
		return out
	}
	if c, ok, _ := snap.ResolveCredentialForCall(providerType, key, overrideName); ok {
		out.ID = c.ID
		out.Name = c.Name
		out.StorageKind = c.StorageKind
		out.IsBYOK = true
	}
	return out
}
