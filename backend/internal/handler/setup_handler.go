package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
)

// SetupHandler exposes the first-run wizard endpoints.
type SetupHandler struct {
	svc service.SetupService
}

// NewSetupHandler creates a SetupHandler backed by the given service.
func NewSetupHandler(svc service.SetupService) *SetupHandler {
	return &SetupHandler{svc: svc}
}

// Status godoc
// @Summary      Check whether first-run setup is required
// @Description  Returns true when no admin account exists (fresh install). Public — no auth required.
// @Tags         setup
// @Produce      json
// @Success      200 {object} dto.SetupStatusResponse
// @Failure      500 {object} apierror.APIError
// @Router       /setup/status [get]
func (h *SetupHandler) Status(c echo.Context) error {
	required, err := h.svc.SetupRequired(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, dto.SetupStatusResponse{SetupRequired: required})
}

// CreateAdmin godoc
// @Summary      Create the initial admin account
// @Description  Creates the first admin user and returns a JWT. Public — no auth required. Rate-limited.
// @Tags         setup
// @Accept       json
// @Produce      json
// @Param        body body dto.SetupAdminRequest true "Admin account payload"
// @Success      201 {object} dto.AuthResponse
// @Failure      400 {object} apierror.APIError "Validation error or password policy violation"
// @Failure      409 {object} apierror.APIError "Admin already exists"
// @Failure      429 {object} apierror.APIError "Rate limit exceeded"
// @Router       /setup/admin [post]
func (h *SetupHandler) CreateAdmin(c echo.Context) error {
	var req dto.SetupAdminRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	resp, err := h.svc.CreateAdmin(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrSetupComplete) {
			return echo.NewHTTPError(http.StatusConflict, "Setup is already complete. Please log in.")
		}
		return err
	}

	return c.JSON(http.StatusCreated, resp)
}

// CreateUser godoc
// @Summary      Create an optional team member during wizard step 2
// @Description  Creates a non-admin user. Requires the JWT issued by POST /setup/admin.
// @Tags         setup
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body dto.SetupUserRequest true "Team member payload"
// @Success      201 {object} dto.SetupUserResponse
// @Failure      400 {object} apierror.APIError "Validation error"
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      409 {object} apierror.APIError "Email already registered"
// @Failure      429 {object} apierror.APIError "Rate limit exceeded"
// @Router       /setup/user [post]
func (h *SetupHandler) CreateUser(c echo.Context) error {
	var req dto.SetupUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	resp, err := h.svc.CreateUser(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return echo.NewHTTPError(http.StatusConflict, "Email already registered.")
		}
		return err
	}

	return c.JSON(http.StatusCreated, resp)
}

// SeedData godoc
// @Summary      Import example seed data (wizard step 4)
// @Description  Populates the workspace with a demo project, sprint, issues, and labels.
// @Description  Idempotent — returns skipped:true if seed data already exists.
// @Tags         setup
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} dto.SetupSeedResponse
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      500 {object} apierror.APIError "Seed failed"
// @Router       /setup/seed [post]
func (h *SetupHandler) SeedData(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	resp, err := h.svc.SeedData(c.Request().Context(), callerID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "seed failed: "+err.Error())
	}
	return c.JSON(http.StatusOK, resp)
}

// CreateProject godoc
// @Summary      Create an optional first project during wizard step 3
// @Description  Creates a project and optionally adds a team member. Requires the JWT from step 1.
// @Tags         setup
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body dto.SetupProjectRequest true "Project payload"
// @Success      201 {object} dto.SetupProjectResponse
// @Failure      400 {object} apierror.APIError "Validation error"
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      404 {object} apierror.APIError "add_user_id refers to non-existent user"
// @Router       /setup/project [post]
func (h *SetupHandler) CreateProject(c echo.Context) error {
	var req dto.SetupProjectRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	callerID := c.Get("userID").(uint)
	resp, err := h.svc.CreateProject(c.Request().Context(), callerID, req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, resp)
}
