package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
	"github.com/sharique/mansooba/pkg/logger"
	"go.uber.org/zap"
)

// AttachmentHandler exposes attachment CRUD at /issues/:id/attachments.
type AttachmentHandler struct {
	svc service.AttachmentService
}

func NewAttachmentHandler(svc service.AttachmentService) *AttachmentHandler {
	return &AttachmentHandler{svc: svc}
}

// Upload godoc
// @Summary      Upload one or more files to an issue
// @Tags         attachments
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        id    path int  true "Issue ID"
// @Param        files formData file true "One or more files (repeat the field for each file)"
// @Success      200 {object} dto.AttachmentUploadResult
// @Failure      403 {object} apierror.APIError
// @Failure      404 {object} apierror.APIError
// @Failure      409 {object} apierror.APIError
// @Router       /issues/{id}/attachments [post]
func (h *AttachmentHandler) Upload(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	issueID, err := parseUintParam(c, "id")
	if err != nil {
		return echo.ErrBadRequest
	}

	form, err := c.MultipartForm()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "could not parse multipart form")
	}
	headers := form.File["files"]
	if len(headers) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "at least one file is required")
	}

	files := make([]dto.AttachmentUploadFile, 0, len(headers))
	for _, fh := range headers {
		f, err := fh.Open()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "could not read uploaded file")
		}
		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "could not read uploaded file")
		}
		files = append(files, dto.AttachmentUploadFile{
			Filename:    fh.Filename,
			Data:        data,
			ContentType: fh.Header.Get("Content-Type"),
		})
	}

	result, err := h.svc.Upload(c.Request().Context(), issueID, callerID, files)
	if err != nil {
		logger.Logger.Info("attachment upload failed", zap.Uint("issueID", issueID), zap.Uint("callerID", callerID), zap.Error(err))
		return mapAttachmentError(err)
	}

	logger.Logger.Info("attachments uploaded", zap.Uint("issueID", issueID), zap.Uint("callerID", callerID),
		zap.Int("uploaded", len(result.Uploaded)), zap.Int("rejected", len(result.Rejected)))
	return c.JSON(http.StatusOK, result)
}

// List godoc
// @Summary      List an issue's attachments
// @Tags         attachments
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Issue ID"
// @Success      200 {object} dto.AttachmentListResponse
// @Failure      403 {object} apierror.APIError
// @Failure      404 {object} apierror.APIError
// @Router       /issues/{id}/attachments [get]
func (h *AttachmentHandler) List(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	issueID, err := parseUintParam(c, "id")
	if err != nil {
		return echo.ErrBadRequest
	}

	attachments, err := h.svc.List(c.Request().Context(), issueID, callerID)
	if err != nil {
		return mapAttachmentError(err)
	}

	resp := dto.AttachmentListResponse{Attachments: make([]dto.AttachmentResponse, 0, len(attachments))}
	for _, a := range attachments {
		resp.Attachments = append(resp.Attachments, *a)
	}
	return c.JSON(http.StatusOK, resp)
}

// Download godoc
// @Summary      Get a presigned download URL for an attachment
// @Tags         attachments
// @Produce      json
// @Security     BearerAuth
// @Param        id  path int true "Issue ID"
// @Param        aid path int true "Attachment ID"
// @Success      200 {object} dto.AttachmentDownloadResponse
// @Failure      403 {object} apierror.APIError
// @Failure      404 {object} apierror.APIError
// @Router       /issues/{id}/attachments/{aid}/download [get]
//
// Returns the presigned URL as JSON rather than a 302 redirect: this
// endpoint requires the normal Authorization: Bearer header, which only a
// same-origin fetch can attach — a plain browser navigation (<a href>)
// can't set custom headers, and a script-initiated fetch can't read a
// cross-origin redirect's Location header either. The client fetches this
// JSON (authenticated, same-origin — no CORS involved) and then navigates
// directly to the returned S3 URL, which needs no Authorization header at
// all: the presigned signature *is* the authorization (research.md Decision 2).
func (h *AttachmentHandler) Download(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	issueID, err := parseUintParam(c, "id")
	if err != nil {
		return echo.ErrBadRequest
	}
	aid, err := parseUintParam(c, "aid")
	if err != nil {
		return echo.ErrBadRequest
	}

	url, filename, err := h.svc.GenerateDownloadURL(c.Request().Context(), issueID, aid, callerID)
	if err != nil {
		return mapAttachmentError(err)
	}
	return c.JSON(http.StatusOK, dto.AttachmentDownloadResponse{URL: url, Filename: filename})
}

// Delete godoc
// @Summary      Delete an attachment
// @Tags         attachments
// @Security     BearerAuth
// @Param        id  path int true "Issue ID"
// @Param        aid path int true "Attachment ID"
// @Success      200
// @Failure      403 {object} apierror.APIError
// @Failure      404 {object} apierror.APIError
// @Router       /issues/{id}/attachments/{aid} [delete]
func (h *AttachmentHandler) Delete(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	issueID, err := parseUintParam(c, "id")
	if err != nil {
		return echo.ErrBadRequest
	}
	aid, err := parseUintParam(c, "aid")
	if err != nil {
		return echo.ErrBadRequest
	}

	if err := h.svc.Delete(c.Request().Context(), issueID, aid, callerID); err != nil {
		logger.Logger.Info("attachment delete failed", zap.Uint("issueID", issueID), zap.Uint("attachmentID", aid), zap.Error(err))
		return mapAttachmentError(err)
	}

	logger.Logger.Info("attachment deleted", zap.Uint("issueID", issueID), zap.Uint("attachmentID", aid), zap.Uint("callerID", callerID))
	return c.JSON(http.StatusOK, map[string]any{})
}

func mapAttachmentError(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrForbidden):
		return echo.NewHTTPError(http.StatusForbidden, "forbidden")
	case errors.Is(err, domain.ErrAttachmentCapReached):
		return echo.NewHTTPError(http.StatusConflict, "attachment limit reached for this issue")
	case errors.Is(err, domain.ErrAttachmentStorageUnavailable):
		return echo.NewHTTPError(http.StatusBadGateway, "storage temporarily unavailable, please try again")
	}
	return err
}
