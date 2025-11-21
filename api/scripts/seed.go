package main

import (
	"encoding/json"
	"finopsbridge/api/internal/database_"
	"finopsbridge/api/internal/models_"
	"finopsbridge/api/internal/policygen_"
	"log"
	"os"
	"time"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@localhost:5432/finopsbridge?sslmode=disable"
	}

	db, err := database_.Initialize(dbURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Create a test organization
	org := models_.Organization{
		ClerkOrgID: "org_test_123",
		Name:       "Test Organization",
	}
		if err := db.FirstOrCreate(&org, models_.Organization{ClerkOrgID: org.ClerkOrgID}).Error; err != nil {
		log.Fatalf("Failed to create organization: %v", err)
	}

	// Create 3 example policies
	policies := []struct {
		name        string
		description string
		policyType  string
		config      map[string]interface{}
	}{
		{
			name:        "Max Monthly Spend - Production",
			description: "Limit production account spending to $5000/month",
			policyType:  "max_spend",
			config: map[string]interface{}{
				"maxAmount": 5000.0,
				"accountId": "prod-account",
			},
		},
		{
			name:        "Block X-Large Instances",
			description: "Prevent deployment of x-large or larger instances",
			policyType:  "block_instance_type",
			config: map[string]interface{}{
				"maxSize": "large",
			},
		},
		{
			name:        "Auto-Stop Idle Resources",
			description: "Automatically stop resources idle for more than 24 hours",
			policyType:  "auto_stop_idle",
			config: map[string]interface{}{
				"idleHours": 24,
			},
		},
	}

	for _, p := range policies {
		rego, err := policygen_.GenerateRego(p.policyType, p.config)
		if err != nil {
			log.Printf("Failed to generate Rego for %s: %v", p.name, err)
			continue
		}

		configJSON, _ := json.Marshal(p.config)

		policy := models_.Policy{
			OrganizationID: org.ID,
			Name:           p.name,
			Description:    p.description,
			Type:           p.policyType,
			Enabled:        true,
			Rego:           rego,
			Config:         string(configJSON),
		}

		if err := db.Create(&policy).Error; err != nil {
			log.Printf("Failed to create policy %s: %v", p.name, err)
		} else {
			log.Printf("Created policy: %s", p.name)
		}
	}

	log.Println("Seed data created successfully!")
}

