package domain

import (
	"encoding/json"
	"time"
)

type User struct {
	ID        string          `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	TenantID  string          `gorm:"type:uuid;not null" json:"tenant_id"`
	Email     string          `gorm:"type:text;not null;unique" json:"email"`
	Name      string          `gorm:"type:text;not null" json:"name"`
	Roles     []string        `gorm:"type:text[];not null;default:'{user}'" json:"roles"`
	Active    bool            `gorm:"not null;default:true" json:"active"`
	Metadata  json.RawMessage `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt time.Time       `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time       `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"updated_at"`
	Tenant    *Tenant         `gorm:"foreignKey:TenantID" json:"-"`
}

func (User) TableName() string {
	return "users"
}

type UserFilter struct {
	TenantID string   `json:"tenant_id"`
	Email    string   `json:"email"`
	Name     string   `json:"name"`
	Roles    []string `json:"roles"`
	Active   *bool    `json:"active"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
	Limit    int      `json:"limit"`
	Offset   int      `json:"offset"`
}
