package models

type LegalSection struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	Content  string `json:"content"`
	Category string `json:"category"` // e.g., "User", "Provider"
	Version  string `json:"version"`  // e.g., "v1.0"
	Updated  string `json:"updated"`  // ISO8601 timestamp
}

const (
	RoleUser     = "User"
	RoleProvider = "Provider"
	RoleBoth     = "Both"
)
