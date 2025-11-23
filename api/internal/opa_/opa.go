package opa

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/open-policy-agent/opa/rego"
)

type Engine struct {
	dir      string
	policies map[string]string // policyID -> rego code
	mu       sync.RWMutex
}

func Initialize(policyDir string) (*Engine, error) {
	// Create policy directory if it doesn't exist
	if err := os.MkdirAll(policyDir, 0755); err != nil {
		return nil, err
	}

	engine := &Engine{
		dir:      policyDir,
		policies: make(map[string]string),
	}

	// Load existing policies from disk
	engine.loadPoliciesFromDisk()

	return engine, nil
}

func (e *Engine) loadPoliciesFromDisk() {
	files, err := os.ReadDir(e.dir)
	if err != nil {
		fmt.Printf("Error reading policy directory: %v\n", err)
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".rego" {
			content, err := os.ReadFile(filepath.Join(e.dir, file.Name()))
			if err != nil {
				fmt.Printf("Error reading policy file %s: %v\n", file.Name(), err)
				continue
			}
			// Extract policy ID from filename (remove .rego extension)
			policyID := file.Name()[:len(file.Name())-5]
			e.policies[policyID] = string(content)
		}
	}
}

func (e *Engine) ReloadPolicies() error {
	e.loadPoliciesFromDisk()
	return nil
}

func (e *Engine) EvaluatePolicy(policyName string, input map[string]interface{}) (bool, map[string]interface{}, error) {
	e.mu.RLock()
	regoCode, exists := e.policies[policyName]
	e.mu.RUnlock()

	if !exists {
		// Try to load from disk
		filename := filepath.Join(e.dir, fmt.Sprintf("%s.rego", policyName))
		content, err := os.ReadFile(filename)
		if err != nil {
			return true, map[string]interface{}{"allow": true, "msg": "policy not found"}, nil
		}
		regoCode = string(content)
		e.mu.Lock()
		e.policies[policyName] = regoCode
		e.mu.Unlock()
	}

	ctx := context.Background()

	// Create a new Rego query to evaluate the "allow" rule
	query, err := rego.New(
		rego.Query("data.finopsbridge.policies.allow"),
		rego.Module(policyName+".rego", regoCode),
	).PrepareForEval(ctx)

	if err != nil {
		return true, map[string]interface{}{"allow": true, "error": err.Error()}, fmt.Errorf("failed to prepare policy: %w", err)
	}

	// Evaluate the policy with the input
	results, err := query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return true, map[string]interface{}{"allow": true, "error": err.Error()}, fmt.Errorf("failed to evaluate policy: %w", err)
	}

	// Check if the policy allows the action
	allowed := false
	if len(results) > 0 && len(results[0].Expressions) > 0 {
		if val, ok := results[0].Expressions[0].Value.(bool); ok {
			allowed = val
		}
	}

	// Also check for violations
	violationQuery, err := rego.New(
		rego.Query("data.finopsbridge.policies.violation"),
		rego.Module(policyName+".rego", regoCode),
	).PrepareForEval(ctx)

	if err == nil {
		violationResults, err := violationQuery.Eval(ctx, rego.EvalInput(input))
		if err == nil && len(violationResults) > 0 && len(violationResults[0].Expressions) > 0 {
			// If violation is true, set allowed to false
			if val, ok := violationResults[0].Expressions[0].Value.(bool); ok && val {
				allowed = false
			}
		}
	}

	result := map[string]interface{}{
		"allow": allowed,
	}

	// Try to get violation message if not allowed
	if !allowed {
		msgQuery, err := rego.New(
			rego.Query("data.finopsbridge.policies.msg"),
			rego.Module(policyName+".rego", regoCode),
		).PrepareForEval(ctx)

		if err == nil {
			msgResults, err := msgQuery.Eval(ctx, rego.EvalInput(input))
			if err == nil && len(msgResults) > 0 && len(msgResults[0].Expressions) > 0 {
				if msg, ok := msgResults[0].Expressions[0].Value.(string); ok {
					result["msg"] = msg
				}
			}
		}
	}

	return allowed, result, nil
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

// SavePolicy saves a Rego policy to disk and updates the in-memory cache
func (e *Engine) SavePolicy(name string, regoCode string) error {
	filename := filepath.Join(e.dir, fmt.Sprintf("%s.rego", name))
	if err := os.WriteFile(filename, []byte(regoCode), 0644); err != nil {
		return err
	}

	// Update in-memory cache
	e.mu.Lock()
	e.policies[name] = regoCode
	e.mu.Unlock()

	return nil
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

