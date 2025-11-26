package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID             string `gorm:"primaryKey"`
	ClerkUserID    string `gorm:"uniqueIndex;not null"`
	Email          string
	Name           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Organizations  []Organization `gorm:"many2many:user_organizations;"`
}

type Organization struct {
	ID            string `gorm:"primaryKey"`
	ClerkOrgID    string `gorm:"uniqueIndex;not null"`
	Name          string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Users         []User           `gorm:"many2many:user_organizations;"`
	CloudProviders []CloudProvider `gorm:"foreignKey:OrganizationID"`
	Policies      []Policy         `gorm:"foreignKey:OrganizationID"`
}

type CloudProvider struct {
	ID             string `gorm:"primaryKey"`
	OrganizationID string `gorm:"index;not null"`
	Type           string `gorm:"not null"` // aws, azure, gcp
	Name           string `gorm:"not null"`
	AccountID      string
	SubscriptionID string
	ProjectID      string
	Status         string `gorm:"default:disconnected"` // connected, disconnected, error
	Credentials    string `gorm:"type:text"`            // JSON encrypted credentials
	MonthlySpend   float64
	ConnectedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Policy struct {
	ID             string `gorm:"primaryKey"`
	OrganizationID string `gorm:"index;not null"`
	Name           string `gorm:"not null"`
	Description    string
	Type           string `gorm:"not null"` // max_spend, block_instance_type, auto_stop_idle, require_tags
	Enabled        bool   `gorm:"default:true"`
	Rego           string `gorm:"type:text;not null"`
	Config         string `gorm:"type:text"` // JSON config
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Violations     []PolicyViolation `gorm:"foreignKey:PolicyID"`
}

type PolicyViolation struct {
	ID            string `gorm:"primaryKey"`
	PolicyID      string `gorm:"index;not null"`
	ResourceID    string `gorm:"not null"`
	ResourceType  string `gorm:"not null"`
	CloudProvider string `gorm:"not null"`
	Message       string `gorm:"type:text"`
	Severity      string `gorm:"default:medium"` // low, medium, high, critical
	Status        string `gorm:"default:pending"` // pending, remediated, ignored
	CreatedAt     time.Time
	RemediatedAt  *time.Time
}

type ActivityLog struct {
	ID        string `gorm:"primaryKey"`
	OrganizationID string `gorm:"index;not null"`
	Type      string `gorm:"not null"` // policy_violation, remediation, policy_created, etc.
	Message   string `gorm:"type:text;not null"`
	Metadata  string `gorm:"type:text"` // JSON metadata
	CreatedAt time.Time
}

type WaitlistEntry struct {
	ID        string `gorm:"primaryKey"`
	Email     string `gorm:"uniqueIndex;not null"`
	Name      string
	Company   string
	CreatedAt time.Time
}

type Webhook struct {
	ID             string `gorm:"primaryKey"`
	OrganizationID string `gorm:"index;not null"`
	Type           string `gorm:"not null"` // slack, discord, teams
	URL            string `gorm:"not null"`
	Enabled        bool   `gorm:"default:true"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type PolicyCategory struct {
	ID          string `gorm:"primaryKey"`
	Name        string `gorm:"not null;uniqueIndex"`
	Description string
	Icon        string
	SortOrder   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Templates   []PolicyTemplate `gorm:"foreignKey:CategoryID"`
}

type PolicyTemplate struct {
	ID                  string `gorm:"primaryKey"`
	CategoryID          string `gorm:"index;not null"`
	Name                string `gorm:"not null"`
	Description         string `gorm:"type:text"`
	PolicyType          string `gorm:"not null"` // max_spend, auto_stop_idle, etc.
	DefaultConfig       string `gorm:"type:text"` // JSON default parameters
	RegoTemplate        string `gorm:"type:text;not null"` // OPA Rego template
	EstimatedSavings    string // e.g., "15-30%", "$5K-20K/month"
	Difficulty          string `gorm:"default:easy"` // easy, medium, hard
	RequiredPermissions string `gorm:"type:text"` // JSON array of required cloud permissions
	Tags                string `gorm:"type:text"` // JSON array of tags
	CloudProviders      string `gorm:"type:text"` // JSON array: ["aws", "azure", "gcp", "oci", "ibm"]
	ComplianceFrameworks string `gorm:"type:text"` // JSON array: ["soc2", "hipaa", "pci-dss"]
	BusinessImpact      string `gorm:"type:text"` // Description of business value
	UsageCount          int    `gorm:"default:0"` // Track popularity
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type PolicyRecommendation struct {
	ID                string `gorm:"primaryKey"`
	OrganizationID    string `gorm:"index;not null"`
	PolicyTemplateID  string `gorm:"index;not null"`
	Status            string `gorm:"default:pending"` // pending, accepted, rejected, deployed
	ConfidenceScore   float64 // 0.0 to 1.0
	EstimatedMonthlySavings float64
	RecommendationReason string `gorm:"type:text"` // AI-generated explanation
	DetectedIssues    string `gorm:"type:text"` // JSON array of detected waste/issues
	SuggestedConfig   string `gorm:"type:text"` // JSON suggested policy configuration
	Priority          string `gorm:"default:medium"` // low, medium, high, critical
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeployedAt        *time.Time
	RejectedAt        *time.Time
	RejectionReason   string
}

type PolicyAdoptionMetrics struct {
	ID                  string `gorm:"primaryKey"`
	OrganizationID      string `gorm:"index;not null"`
	PolicyID            string `gorm:"index;not null"`
	Month               string `gorm:"not null"` // YYYY-MM format
	ViolationCount      int
	RemediationCount    int
	CostSavings         float64
	ResourcesAffected   int
	ComplianceScore     float64 // 0.0 to 100.0
	AverageRemediationTime int // seconds
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// BeforeCreate hooks
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = generateID()
	}
	return nil
}

func (o *Organization) BeforeCreate(tx *gorm.DB) error {
	if o.ID == "" {
		o.ID = generateID()
	}
	return nil
}

func (cp *CloudProvider) BeforeCreate(tx *gorm.DB) error {
	if cp.ID == "" {
		cp.ID = generateID()
	}
	return nil
}

func (p *Policy) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = generateID()
	}
	return nil
}

func (pv *PolicyViolation) BeforeCreate(tx *gorm.DB) error {
	if pv.ID == "" {
		pv.ID = generateID()
	}
	return nil
}

func (al *ActivityLog) BeforeCreate(tx *gorm.DB) error {
	if al.ID == "" {
		al.ID = generateID()
	}
	return nil
}

func (we *WaitlistEntry) BeforeCreate(tx *gorm.DB) error {
	if we.ID == "" {
		we.ID = generateID()
	}
	return nil
}

func (w *Webhook) BeforeCreate(tx *gorm.DB) error {
	if w.ID == "" {
		w.ID = generateID()
	}
	return nil
}

func (pc *PolicyCategory) BeforeCreate(tx *gorm.DB) error {
	if pc.ID == "" {
		pc.ID = generateID()
	}
	return nil
}

func (pt *PolicyTemplate) BeforeCreate(tx *gorm.DB) error {
	if pt.ID == "" {
		pt.ID = generateID()
	}
	return nil
}

func (pr *PolicyRecommendation) BeforeCreate(tx *gorm.DB) error {
	if pr.ID == "" {
		pr.ID = generateID()
	}
	return nil
}

func (pam *PolicyAdoptionMetrics) BeforeCreate(tx *gorm.DB) error {
	if pam.ID == "" {
		pam.ID = generateID()
	}
	return nil
}

func generateID() string {
	return time.Now().Format("20060102150405") + randomString(8)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}

