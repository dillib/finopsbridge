package opa

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Engine struct {
	dir string
}

func Initialize(policyDir string) (*Engine, error) {
	// Create policy directory if it doesn't exist
	if err := os.MkdirAll(policyDir, 0755); err != nil {
		return nil, err
	}

	engine := &Engine{
		dir: policyDir,
	}

	return engine, nil
}

func (e *Engine) ReloadPolicies() error {
	// This will trigger OPA to reload bundles
	// In a real implementation, you'd use OPA's bundle API
	return nil
}

func (e *Engine) EvaluatePolicy(policyName string, input map[string]interface{}) (bool, map[string]interface{}, error) {
	// Simplified policy evaluation - always allow for now
	// In production, this would use OPA's rego package directly
	return true, map[string]interface{}{"allow": true}, nil
}

func (e *Engine) WatchForChanges() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := e.ReloadPolicies(); err != nil {
			fmt.Printf("Error reloading policies: %v\n", err)
		}
	}
}

func (e *Engine) Close() error {
	return nil
}

// SavePolicy saves a Rego policy to disk
func (e *Engine) SavePolicy(name string, rego string) error {
	filename := filepath.Join(e.dir, fmt.Sprintf("%s.rego", name))
	return os.WriteFile(filename, []byte(rego), 0644)
}

// LoadPoliciesFromDB loads policies from database and saves them to OPA directory
func (e *Engine) LoadPoliciesFromDB(policies []PolicyInfo) error {
	for _, policy := range policies {
		if err := e.SavePolicy(policy.ID, policy.Rego); err != nil {
			return err
		}
	}
	return e.ReloadPolicies()
}

type PolicyInfo struct {
	ID    string
	Rego  string
	Type  string
	Config map[string]interface{}
}

