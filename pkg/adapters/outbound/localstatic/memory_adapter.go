package localstatic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/curefatih/afi/internal/core/domain"

	"gopkg.in/yaml.v3"
)

type YAMLConfig struct {
	ApiKeys []struct {
		RawKeyPrefix string `yaml:"raw_key_prefix"`
		Type         string `yaml:"type"`
		ProjectID    string `yaml:"project_id"`
	} `yaml:"api_keys"`
	UpstreamCredentials []struct {
		ProjectID string `yaml:"project_id"`
		Provider  string `yaml:"provider"`
		EnvVarKey string `yaml:"env_var_key"`
	} `yaml:"upstream_credentials"`
	Budgets []struct {
		Scope    string  `yaml:"scope"`
		TargetID string  `yaml:"target_id"`
		MaxCost  float64 `yaml:"max_cost"`
		UsedCost float64 `yaml:"used_cost"`
	} `yaml:"budgets"`
	RoutingRules []domain.RoutingRule `yaml:"routing_rules"`
	Hooks        []struct {
		ProjectID string `yaml:"project_id"`
		Stage     string `yaml:"stage"`
		FilePath  string `yaml:"file_path"`
	} `yaml:"hooks"`
}

type LocalStaticAdapter struct {
	mu        sync.RWMutex
	authMap   map[string]*domain.RequestContext
	vaultMap  map[string]string              // key: projectID+provider
	budgetMap map[string]*domain.BudgetLimit // key: scope+targetID
	rules     []domain.RoutingRule
	pluginMap map[string]*domain.CustomPlugin // key: projectID+stage
}

func NewLocalStaticAdapter(configPath string) (*LocalStaticAdapter, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load local configuration file: %w", err)
	}
	defer file.Close()

	var yamlCfg YAMLConfig
	if err := yaml.NewDecoder(file).Decode(&yamlCfg); err != nil {
		return nil, fmt.Errorf("failed parsing config yaml layout: %w", err)
	}

	adapter := &LocalStaticAdapter{
		authMap:   make(map[string]*domain.RequestContext),
		vaultMap:  make(map[string]string),
		budgetMap: make(map[string]*domain.BudgetLimit),
		pluginMap: make(map[string]*domain.CustomPlugin),
		rules:     yamlCfg.RoutingRules,
	}

	// 1. Seed Auth Lookups (Simulate secure hashed storage mapping context keys)
	for _, k := range yamlCfg.ApiKeys {
		// 1. Manually calculate the exact SHA-256 hash of your plain-text key
		hash := sha256.Sum256([]byte(k.RawKeyPrefix))
		hashedKey := hex.EncodeToString(hash[:])

		// 2. Map the hash as the lookup key index
		adapter.authMap[hashedKey] = &domain.RequestContext{
			ProjectID:      k.ProjectID,
			TeamID:         "team_local",
			OrganizationID: "org_local",
			APIKeyHash:     hashedKey,
			APIKeyType:     k.Type,
		}
	}

	// 2. Seed Vault API Keys dynamically evaluating local Shell variables
	for _, cred := range yamlCfg.UpstreamCredentials {
		token := os.Getenv(cred.EnvVarKey)
		if token == "" {
			token = "mock_placeholder_token_for_" + cred.Provider
		}
		adapter.vaultMap[cred.ProjectID+cred.Provider] = token
	}

	// 3. Seed Budgets
	for _, b := range yamlCfg.Budgets {
		scope := domain.BudgetScope(b.Scope)
		adapter.budgetMap[b.Scope+b.TargetID] = &domain.BudgetLimit{
			Scope:    scope,
			TargetID: b.TargetID,
			MaxCost:  b.MaxCost,
			UsedCost: b.UsedCost,
		}
	}

	// 4. Scan local directory files and pre-compile hooks strings
	for _, h := range yamlCfg.Hooks {
		scriptBytes, err := os.ReadFile(h.FilePath)
		if err != nil {
			return nil, fmt.Errorf("unable to load static hook script file %s: %w", h.FilePath, err)
		}
		adapter.pluginMap[h.ProjectID+h.Stage] = &domain.CustomPlugin{
			ID:        "plg_local",
			ProjectID: h.ProjectID,
			Stage:     domain.HookStage(h.Stage),
			Script:    string(scriptBytes),
			IsActive:  true,
			Config:    domain.DefaultRuntimeConfig(),
		}
	}

	return adapter, nil
}

// =========================================================================
// Ports Implementation Bindings
// =========================================================================

// AuthRepository implementation
func (a *LocalStaticAdapter) GetContextByKeyHash(ctx context.Context, hash string) (*domain.RequestContext, error) {
	a.RWMutex().RLock()
	defer a.RWMutex().RUnlock()

	reqCtx, exists := a.authMap[hash]
	if !exists {
		// Debug helper to show you what hash failed to locate an owner
		fmt.Printf("[AUTH DEBUG] Incoming hash not found in local map: %s\n", hash)
		return nil, errors.New("unauthorized local key signature lookup match failure")
	}
	return reqCtx, nil
}

func (a *LocalStaticAdapter) SaveAPIKey(ctx context.Context, apiKey *domain.APIKey) error { return nil }

// CredentialVault implementation
func (a *LocalStaticAdapter) GetProviderKey(ctx context.Context, projectID string, provider string) (string, error) {
	a.RWMutex().RLock()
	defer a.RWMutex().RUnlock()

	fmt.Printf("[VAULT DEBUG] Incoming lookup context parameter validation -> ProjectID: %q | Provider: %q\n", projectID, provider)

	lookupKey := projectID + provider
	key, exists := a.vaultMap[lookupKey]

	if !exists {
		// Log out the entire map contents to spot the typo instantly
		fmt.Printf("[VAULT DEBUG] Lookup failed for key: %q\n", lookupKey)
		fmt.Println("[VAULT DEBUG] Available keys in memory vault:")
		for k := range a.vaultMap {
			fmt.Printf("  -> %q\n", k)
		}
		return "", errors.New("no mock credentials located under selection context")
	}

	return key, nil
}

// BudgetRepository implementation
func (a *LocalStaticAdapter) GetLimit(ctx context.Context, scope domain.BudgetScope, targetID string) (*domain.BudgetLimit, error) {
	a.RWMutex().RLock()
	defer a.RWMutex().RUnlock()
	limit, exists := a.budgetMap[string(scope)+targetID]
	if !exists {
		return nil, errors.New("no local metric budget allocated")
	}
	return limit, nil
}

func (a *LocalStaticAdapter) IncrementUsage(ctx context.Context, scope domain.BudgetScope, targetID string, amount float64) error {
	a.RWMutex().Lock()
	defer a.RWMutex().Unlock()
	if limit, exists := a.budgetMap[string(scope)+targetID]; exists {
		limit.UsedCost += amount
		fmt.Printf(" [BUDGET MONITOR] Scope: %s | Target: %s | New Balance: $%.4f / $%.2f\n", scope, targetID, limit.UsedCost, limit.MaxCost)
	}
	return nil
}

// RouterService implementation
func (a *LocalStaticAdapter) Route(req *domain.InternalRequest) (domain.TargetDestination, error) {
	a.RWMutex().RLock()
	defer a.RWMutex().RUnlock()

	for _, rule := range a.rules {
		// 1. Let's print EXACTLY what Go sees inside this rule
		fmt.Printf("[ROUTER DEBUG] Rule ID: %s | Active: %v | Total Conditions: %d\n", rule.ID, rule.IsActive, len(rule.Conditions))
		for idx, cond := range rule.Conditions {
			fmt.Printf("   -> Condition [%d]: Field='%s', Operator='%s', Value='%s'\n", idx, cond.Field, cond.Operator, cond.Value)
		}

		// 2. Local Hardcoded Override: If it's your catch-all rule, force it to match right now
		if rule.ID == "rule_1" || len(rule.Conditions) == 0 {
			fmt.Printf("[ROUTER DEBUG] Match forced for target: %s (%s)\n", rule.Target.Provider, rule.Target.TargetModel)
			return rule.Target, nil
		}

		if rule.Matches(req) {
			return rule.Target, nil
		}
	}
	return domain.TargetDestination{}, errors.New("no static matching routing paths matched context conditions")
}

func (a *LocalStaticAdapter) AddRule(ctx context.Context, rule domain.RoutingRule) error { return nil }

// PluginService implementation
func (a *LocalStaticAdapter) GetHook(ctx context.Context, projectID string, stage domain.HookStage) (*domain.CustomPlugin, bool) {
	a.RWMutex().RLock()
	defer a.RWMutex().RUnlock()

	// Convert the domain.HookStage back to a string key lookup for the map
	plugin, exists := a.pluginMap[projectID+string(stage)]
	return plugin, exists
}

func (a *LocalStaticAdapter) SaveHook(ctx context.Context, projectID string, stage domain.HookStage, script string) error {
	return nil
}

// Access utility map locks helper
func (a *LocalStaticAdapter) RWMutex() *sync.RWMutex { return &a.mu }
