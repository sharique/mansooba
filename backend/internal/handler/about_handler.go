package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
)

// AboutHandler exposes where this instance's source and licence can be
// found (016-agpl-dual-licensing, AGPL-3.0 section 13).
type AboutHandler struct {
	svc service.InstanceService
}

func NewAboutHandler(svc service.InstanceService) *AboutHandler {
	return &AboutHandler{svc: svc}
}

// Get godoc
// @Summary      This instance's version, source location and licence
// @Description  Public — no auth required. Lets any user of a running instance, including on the sign-in page, find the source they are entitled to under AGPL-3.0 section 13.
// @Tags         about
// @Produce      json
// @Success      200 {object} dto.AboutResponse
// @Router       /about [get]
func (h *AboutHandler) Get(c echo.Context) error {
	info := h.svc.Info()
	c.Response().Header().Set("Cache-Control", "public, max-age=300")
	return c.JSON(http.StatusOK, dto.AboutResponse{
		Version:    info.Version,
		SourceURL:  info.SourceURL,
		License:    info.License,
		LicenseURL: info.LicenseURL,
	})
}
