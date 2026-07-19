package policy

import (
	"fmt"
	"sort"
	"sync"

	"github.com/curefatih/afi/internal/snapshot"
	"github.com/google/cel-go/cel"
)

// Request is the CEL evaluation context for a gateway call.
type Request struct {
	Model  string
	Path   string
	Stream bool
}

// Evaluator compiles and runs boolean CEL allow-policies.
type Evaluator struct {
	env   *cel.Env
	cache sync.Map // expression -> cel.Program
}

// NewEvaluator builds a shared CEL environment.
func NewEvaluator() (*Evaluator, error) {
	env, err := cel.NewEnv(
		cel.Variable("request", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("key", cel.MapType(cel.StringType, cel.DynType)),
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
	_, err = ev.program(expression)
	return err
}

func (e *Evaluator) program(expression string) (cel.Program, error) {
	if expression == "" {
		return nil, fmt.Errorf("empty expression")
	}
	if v, ok := e.cache.Load(expression); ok {
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
	e.cache.Store(expression, prg)
	return prg, nil
}

// Evaluate runs enabled org policies (priority desc, then name). All must be true.
// Returns the first denying policy name when allowed is false.
func (e *Evaluator) Evaluate(policies []snapshot.Policy, key snapshot.APIKey, req Request) (allowed bool, deniedName string, err error) {
	var matched []snapshot.Policy
	for _, p := range policies {
		if !p.Enabled || p.OrganizationID != key.OrganizationID {
			continue
		}
		matched = append(matched, p)
	}
	if len(matched) == 0 {
		return true, "", nil
	}
	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].Priority != matched[j].Priority {
			return matched[i].Priority > matched[j].Priority
		}
		return matched[i].Name < matched[j].Name
	})

	vars := map[string]any{
		"request": map[string]any{
			"model":  req.Model,
			"path":   req.Path,
			"stream": req.Stream,
		},
		"key": map[string]any{
			"id":              key.ID,
			"organization_id": key.OrganizationID,
			"project_id":      key.ProjectID,
			"kind":            key.Kind,
			"owner_user_id":   key.OwnerUserID,
			"name":            key.Name,
		},
	}

	for _, p := range matched {
		prg, err := e.program(p.Expression)
		if err != nil {
			return false, "", fmt.Errorf("policy %q: %w", p.Name, err)
		}
		out, _, err := prg.Eval(vars)
		if err != nil {
			return false, "", fmt.Errorf("policy %q: %w", p.Name, err)
		}
		ok, valid := out.Value().(bool)
		if !valid {
			return false, "", fmt.Errorf("policy %q: non-bool result", p.Name)
		}
		if !ok {
			return false, p.Name, nil
		}
	}
	return true, "", nil
}
