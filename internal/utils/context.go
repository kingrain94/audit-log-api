package utils

import (
	"context"
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

type ContextKey string

const (
	ClaimsKey   ContextKey = "claims"
	TenantIDKey ContextKey = "tenant_id"
)

var (
	ErrNoClaimsInContext   = errors.New("no claims found in context")
	ErrInvalidClaimsType   = errors.New("invalid claims type")
	ErrNoTenantIDInClaims  = errors.New("no tenant_id found in claims")
	ErrInvalidTenantIDType = errors.New("tenant_id must be a string")
)

func GetTenantIDFromContext(c context.Context) (string, error) {
	claims, exists := c.Value(ClaimsKey).(jwt.MapClaims)
	if !exists {
		return "", ErrNoClaimsInContext
	}

	tenantID, exists := claims[string(TenantIDKey)]
	if !exists {
		return "", ErrNoTenantIDInClaims
	}

	tenantIDStr, ok := tenantID.(string)
	if !ok {
		return "", ErrInvalidTenantIDType
	}

	return tenantIDStr, nil
}
