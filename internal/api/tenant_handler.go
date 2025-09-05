package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kingrain94/audit-log-api/internal/api/dto"
	"github.com/kingrain94/audit-log-api/internal/domain"
)

//go:generate mockery --name TenantService --output ../mocks
type TenantService interface {
	Create(ctx context.Context, req dto.CreateTenantRequest) (dto.CreateTenantResponse, error)
	GetByID(ctx context.Context, id string) (*domain.Tenant, error)
	Update(ctx context.Context, tenant *domain.Tenant) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]dto.CreateTenantResponse, error)
}

type TenantHandler struct {
	*BaseHandler
	service TenantService
}

func NewTenantHandler(service TenantService) *TenantHandler {
	return &TenantHandler{service: service}
}

// CreateTenant godoc
// @Summary Create a new tenant
// @Description Create a new tenant with specified configuration
// @Tags tenants
// @Accept json
// @Produce json
// @Param body body dto.CreateTenantRequest true "Tenant object"
// @Success 201 {object} dto.CreateTenantResponse
// @Failure 400 {object} dto.Error
// @Failure 401 {object} dto.Error
// @Failure 500 {object} dto.Error
// @Router /tenants [post]
func (h *TenantHandler) CreateTenant(c *gin.Context) {
	var req dto.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error{Error: err.Error()})
		return
	}

	tenant, err := h.service.Create(h.RequestCtx(c), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tenant)
}

// ListTenants godoc
// @Summary List all tenants
// @Description Get a list of all tenants that the authenticated user has access to
// @Tags tenants
// @Produce json
// @Success 200 {array} dto.CreateTenantResponse
// @Failure 401 {object} dto.Error
// @Failure 500 {object} dto.Error
// @Router /tenants [get]
func (h *TenantHandler) ListTenants(c *gin.Context) {
	tenants, err := h.service.List(h.RequestCtx(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, tenants)
}
