package opa

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/open-policy-agent/opa/sdk"
)

type Engine struct {
	opa *sdk.OPA
	dir string
}

func Initialize(policyDir string) (*Engine, error) {
	// Create policy directory if it doesn't exist
	if err := os.MkdirAll(policyDir, 0755); err != nil {
		return nil, err
	}

	config := []byte(fmt.Sprintf(`{
		"services": {
			"default": {
				"url": "http://localhost:8181"
			}
		},
		"bundles": {
			"finopsbridge": {
				"resource": "file://%s"
			}
		}
	}`, policyDir))

	opa, err := sdk.New(context.Background(), sdk.Options{
		Config: config,
	})
	if err != nil {
		return nil, err
	}

	engine := &Engine{
		opa: opa,
		dir: policyDir,
	}

	// Initial policy load
	if err := engine.ReloadPolicies(); err != nil {
		return nil, err
	}

	return engine, nil
}

func (e *Engine) ReloadPolicies() error {
	// This will trigger OPA to reload bundles
	// In a real implementation, you'd use OPA's bundle API
	return nil
}

func (e *Engine) EvaluatePolicy(policyName string, input map[string]interface{}) (bool, map[string]interface{}, error) {
	ctx := context.Background()

	result, err := e.opa.Decision(ctx, sdk.DecisionOptions{
		Path:  fmt.Sprintf("finopsbridge/policies/%s", policyName),
		Input: input,
	})

	if err != nil {
		return false, nil, err
	}

	allowed, ok := result.Result.(bool)
	if !ok {
		// Try to extract from result map
		if resultMap, ok := result.Result.(map[string]interface{}); ok {
			if a, ok := resultMap["allow"].(bool); ok {
				allowed = a
			}
		}
	}

	return allowed, result.Result.(map[string]interface{}), nil
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
	return e.opa.Close(context.Background())
}

// SavePolicy saves a Rego policy to disk
func (e *Engine) SavePolicy(name string, rego string) error {
	filename := filepath.Join(e.dir, fmt.Sprintf("%s.rego", name))
	return ioutil.WriteFile(filename, []byte(rego), 0644)
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

