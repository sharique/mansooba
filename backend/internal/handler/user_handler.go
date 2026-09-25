package handler

import (
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
	"github.com/sharique/mansooba/pkg/logger"
	"go.uber.org/zap"
)

// UserHandler exposes profile, my-activity, and my-issues endpoints for the current user.
type UserHandler struct {
	userSvc     service.UserService
	activitySvc service.ActivityService
	issueSvc    service.IssueService
}

// NewUserHandler creates a UserHandler backed by the given services.
func NewUserHandler(userSvc service.UserService, activitySvc service.ActivityService, issueSvc service.IssueService) *UserHandler {
	return &UserHandler{userSvc: userSvc, activitySvc: activitySvc, issueSvc: issueSvc}
}

// GetProfile godoc
// @Summary      Get current user profile
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} dto.UserProfileResponse
// @Failure      401 {object} apierror.APIError
// @Router       /auth/me [get]
func (h *UserHandler) GetProfile(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	resp, err := h.userSvc.GetProfile(c.Request().Context(), callerID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

// UpdateProfile godoc
// @Summary      Update current user profile
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body dto.UpdateProfileRequest true "Profile update payload"
// @Success      200 {object} dto.UserProfileResponse
// @Failure      400 {object} apierror.APIError
// @Failure      401 {object} apierror.APIError
// @Router       /auth/me [put]
func (h *UserHandler) UpdateProfile(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	var req dto.UpdateProfileRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	resp, err := h.userSvc.UpdateProfile(c.Request().Context(), callerID, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, resp)
}

// GetMyActivity godoc
// @Summary      Get activity feed for current user
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        limit  query int false "Max results (default 20, max 100)"
// @Param        offset query int false "Pagination offset"
// @Success      200 {array} dto.ActivityEventResponse
// @Failure      401 {object} apierror.APIError
// @Router       /auth/me/activity [get]
func (h *UserHandler) GetMyActivity(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	limit := 20
	offset := 0
	if v := c.QueryParam("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return echo.ErrBadRequest
		}
		limit = n
	}
	if v := c.QueryParam("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return echo.ErrBadRequest
		}
		offset = n
	}
	if limit > 100 {
		limit = 100
	}
	events, err := h.activitySvc.GetMyActivity(c.Request().Context(), callerID, limit, offset)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, events)
}

// UploadAvatar godoc
// @Summary      Upload current user's avatar
// @Tags         users
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        avatar formData file true "Avatar image (JPEG/PNG/WebP, max 2 MB)"
// @Success      200 {object} dto.UserProfileResponse
// @Failure      400 {object} apierror.APIError
// @Failure      401 {object} apierror.APIError
// @Failure      413 {object} apierror.APIError
// @Failure      502 {object} apierror.APIError
// @Router       /auth/me/avatar [post]
func (h *UserHandler) UploadAvatar(c echo.Context) error {
	callerID := c.Get("userID").(uint)

	fileHeader, err := c.FormFile("avatar")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "avatar file is required")
	}

	f, err := fileHeader.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "could not read avatar file")
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "could not read avatar file")
	}

	contentType := fileHeader.Header.Get("Content-Type")
	resp, err := h.userSvc.UploadAvatar(c.Request().Context(), callerID, fileHeader.Filename, data, contentType)
	if err != nil {
		if errors.Is(err, domain.ErrAvatarStorageUnavailable) {
			logger.Logger.Error("avatar upload failed: storage unavailable", zap.Uint("userID", callerID), zap.Error(err))
			return echo.NewHTTPError(http.StatusBadGateway, "avatar storage temporarily unavailable, please retry")
		}
		logger.Logger.Info("avatar upload failed", zap.Uint("userID", callerID), zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	logger.Logger.Info("avatar uploaded", zap.Uint("userID", callerID), zap.Int("bytes", len(data)))
	return c.JSON(http.StatusOK, resp)
}

// DeleteAvatar godoc
// @Summary      Delete current user's avatar
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} dto.UserProfileResponse
// @Failure      401 {object} apierror.APIError
// @Router       /auth/me/avatar [delete]
func (h *UserHandler) DeleteAvatar(c echo.Context) error {
	callerID := c.Get("userID").(uint)

	resp, err := h.userSvc.DeleteAvatar(c.Request().Context(), callerID)
	if err != nil {
		if errors.Is(err, domain.ErrAvatarStorageUnavailable) {
			logger.Logger.Error("avatar delete failed: storage unavailable", zap.Uint("userID", callerID), zap.Error(err))
			return echo.NewHTTPError(http.StatusBadGateway, "avatar storage temporarily unavailable, please retry")
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	logger.Logger.Info("avatar deleted", zap.Uint("userID", callerID))
	return c.JSON(http.StatusOK, resp)
}

// avatarFilenamePattern is the exact shape avatarstorage.Save generates. It is
// matched before any storage key is built so a crafted filename can never
// address an object outside avatars/ (e.g. an attachment).
var avatarFilenamePattern = regexp.MustCompile(`^avatar-\d+\.(jpg|png|webp)$`)

// ServeAvatar godoc
// @Summary      Serve a user avatar image
// @Description  PUBLIC — intentionally unauthenticated, like the /uploads static route it replaces: browser <img> tags cannot send an Authorization header, and avatars carry no per-viewer access restriction (ADR-033).
// @Tags         users
// @Produce      image/jpeg,image/png,image/webp
// @Param        filename path string true "avatar-{userID}.{jpg|png|webp}"
// @Success      200 {file} binary
// @Failure      404 {object} apierror.APIError
// @Router       /avatars/{filename} [get]
func (h *UserHandler) ServeAvatar(c echo.Context) error {
	filename := c.Param("filename")
	if !avatarFilenamePattern.MatchString(filename) {
		return echo.NewHTTPError(http.StatusNotFound, "avatar not found")
	}

	body, contentType, err := h.userSvc.GetAvatar(c.Request().Context(), filename)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "avatar not found")
		}
		logger.Logger.Error("avatar serve failed", zap.String("filename", filename), zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "could not load avatar")
	}
	defer body.Close()

	// The stored URL carries a ?v= cache-buster that changes on every upload,
	// so a moderate max-age only avoids refetching an unchanged URL.
	c.Response().Header().Set("Cache-Control", "public, max-age=3600")
	return c.Stream(http.StatusOK, contentType, body)
}

// GetMyIssues godoc
// @Summary      Get issues assigned to the current user
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        status query string false "Filter by status (todo, in_progress, in_review, done, backlog)"
// @Success      200 {array} dto.IssueResponse
// @Failure      401 {object} apierror.APIError
// @Router       /auth/me/issues [get]
func (h *UserHandler) GetMyIssues(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	q := dto.IssueListQuery{
		Status: c.QueryParam("status"),
	}
	issues, err := h.issueSvc.GetMyIssues(c.Request().Context(), callerID, q)
	if err != nil {
		return err
	}
	if issues == nil {
		issues = []*dto.IssueResponse{}
	}
	return c.JSON(http.StatusOK, issues)
}
