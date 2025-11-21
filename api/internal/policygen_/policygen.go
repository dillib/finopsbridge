package policygen

import (
	"fmt"
)

func GenerateRego(policyType string, config map[string]interface{}) (string, error) {
	switch policyType {
	case "max_spend":
		return generateMaxSpendPolicy(config), nil
	case "block_instance_type":
		return generateBlockInstanceTypePolicy(config), nil
	case "auto_stop_idle":
		return generateAutoStopIdlePolicy(config), nil
	case "require_tags":
		return generateRequireTagsPolicy(config), nil
	default:
		return "", fmt.Errorf("unknown policy type: %s", policyType)
	}
}

func generateMaxSpendPolicy(config map[string]interface{}) string {
	maxAmount := config["maxAmount"]
	accountId := config["accountId"]

	accountFilter := ""
	if accountId != nil && accountId != "" {
		accountFilter = fmt.Sprintf(`input.account_id == "%s" &&`, accountId)
	}

	return fmt.Sprintf(`package finopsbridge.policies.max_spend

default allow = false

allow {
	%s
	input.monthly_spend <= %v
}

violation {
	%s
	input.monthly_spend > %v
	msg := sprintf("Monthly spend $%v exceeds limit of $%v", [input.monthly_spend, %v])
}`, accountFilter, maxAmount, accountFilter, maxAmount, maxAmount, maxAmount, maxAmount)
}

func generateBlockInstanceTypePolicy(config map[string]interface{}) string {
	maxSize := config["maxSize"]
	
	sizeMap := map[string]int{
		"small":  1,
		"medium": 2,
		"large":  3,
		"xlarge": 4,
	}
	
	maxSizeValue := sizeMap[maxSize.(string)]

	return fmt.Sprintf(`package finopsbridge.policies.block_instance_type

default allow = true

allow {
	input.instance_size <= %d
}

violation {
	input.instance_size > %d
	msg := sprintf("Instance size exceeds maximum allowed size: %s", [input.instance_size])
}`, maxSizeValue, maxSizeValue, maxSize)
}

func generateAutoStopIdlePolicy(config map[string]interface{}) string {
	idleHours := config["idleHours"]

	return fmt.Sprintf(`package finopsbridge.policies.auto_stop_idle

default allow = true

violation {
	input.idle_hours >= %v
	msg := sprintf("Resource has been idle for %v hours, should be stopped", [input.idle_hours])
}`, idleHours, idleHours)
}

func generateRequireTagsPolicy(config map[string]interface{}) string {
	requiredTags := config["requiredTags"]
	tagsList := ""
	
	if tags, ok := requiredTags.([]interface{}); ok {
		for i, tag := range tags {
			if i > 0 {
				tagsList += ", "
			}
			tagsList += fmt.Sprintf(`"%s"`, tag)
		}
	}

	return fmt.Sprintf(`package finopsbridge.policies.require_tags

default allow = true

violation {
	missing_tag := required_tags[_]
	not input.tags[missing_tag]
	msg := sprintf("Missing required tag: %s", [missing_tag])
}

required_tags = [%s]`, tagsList)
}

