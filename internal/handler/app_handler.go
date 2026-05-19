package handler

import (
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"sso.pelajarnumagetan.or.id/internal/middleware"
	"sso.pelajarnumagetan.or.id/internal/service"
	"sso.pelajarnumagetan.or.id/internal/utils"
)

type AppHandler struct {
	appService service.AppService
}

func NewAppHandler(appService service.AppService) *AppHandler {
	return &AppHandler{appService: appService}
}

// Create — POST /v1/apps
func (h *AppHandler) Create(c echo.Context) error {
	claims := getClaims(c)
	if claims == nil {
		return utils.Unauthorized(c, "Unauthorized")
	}

	var req service.CreateAppRequest
	if err := c.Bind(&req); err != nil {
		return utils.BadRequest(c, "Request tidak valid")
	}
	if err := c.Validate(&req); err != nil {
		if he, ok := err.(*echo.HTTPError); ok {
			return utils.BadRequest(c, he.Message.(string))
		}
		return utils.BadRequest(c, "Validasi gagal")
	}

	resp, err := h.appService.Create(claims.UserID, &req)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.Created(c, "Aplikasi berhasil didaftarkan. Simpan client_secret ini — tidak akan ditampilkan lagi!", resp)
}

// GetMyApps — GET /v1/apps
func (h *AppHandler) GetMyApps(c echo.Context) error {
	claims := getClaims(c)
	if claims == nil {
		return utils.Unauthorized(c, "Unauthorized")
	}

	apps, err := h.appService.GetByOwner(claims.UserID)
	if err != nil {
		return utils.InternalError(c, err.Error())
	}

	return utils.OK(c, "Daftar aplikasi", apps)
}

// GetByID — GET /v1/apps/:id
func (h *AppHandler) GetByID(c echo.Context) error {
	claims := getClaims(c)
	if claims == nil {
		return utils.Unauthorized(c, "Unauthorized")
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.BadRequest(c, "ID tidak valid")
	}

	app, err := h.appService.GetByID(id, claims.UserID)
	if err != nil {
		return utils.NotFound(c, err.Error())
	}

	return utils.OK(c, "Detail aplikasi", app)
}

// Update — PUT /v1/apps/:id
func (h *AppHandler) Update(c echo.Context) error {
	claims := getClaims(c)
	if claims == nil {
		return utils.Unauthorized(c, "Unauthorized")
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.BadRequest(c, "ID tidak valid")
	}

	var req service.UpdateAppRequest
	if err := c.Bind(&req); err != nil {
		return utils.BadRequest(c, "Request tidak valid")
	}

	app, err := h.appService.Update(id, claims.UserID, &req)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.OK(c, "Aplikasi berhasil diperbarui", app)
}

// Delete — DELETE /v1/apps/:id
func (h *AppHandler) Delete(c echo.Context) error {
	claims := getClaims(c)
	if claims == nil {
		return utils.Unauthorized(c, "Unauthorized")
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.BadRequest(c, "ID tidak valid")
	}

	if err := h.appService.Delete(id, claims.UserID); err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.OK(c, "Aplikasi berhasil dihapus", nil)
}

// RegenerateSecret — POST /v1/apps/:id/regenerate
func (h *AppHandler) RegenerateSecret(c echo.Context) error {
	claims := getClaims(c)
	if claims == nil {
		return utils.Unauthorized(c, "Unauthorized")
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.BadRequest(c, "ID tidak valid")
	}

	secret, err := h.appService.RegenerateSecret(id, claims.UserID)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.OK(c, "Secret berhasil digenerate ulang. Simpan — tidak akan ditampilkan lagi!", map[string]string{
		"client_secret": secret,
	})
}

// ToggleActive — PUT /v1/apps/:id/toggle-active
func (h *AppHandler) ToggleActive(c echo.Context) error {
	claims := getClaims(c)
	if claims == nil {
		return utils.Unauthorized(c, "Unauthorized")
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.BadRequest(c, "ID tidak valid")
	}

	app, err := h.appService.ToggleActive(id, claims.UserID)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}

	status := "dinonaktifkan"
	if app.IsActive {
		status = "diaktifkan"
	}
	return utils.OK(c, "Aplikasi berhasil "+status, app)
}

// GetPublicInfo — GET /v1/apps/:id/info (public, tanpa auth)
func (h *AppHandler) GetPublicInfo(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.BadRequest(c, "ID tidak valid")
	}

	app, err := h.appService.GetPublicInfo(id)
	if err != nil {
		return utils.NotFound(c, err.Error())
	}

	return utils.OK(c, "Info aplikasi", app)
}

// ===== ADMIN HANDLERS =====

// AdminGetAll — GET /v1/admin/apps
func (h *AppHandler) AdminGetAll(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if perPage < 1 {
		perPage = 10
	}
	status := c.QueryParam("status")

	apps, total, err := h.appService.GetAll(page, perPage, status)
	if err != nil {
		return utils.InternalError(c, err.Error())
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"message": "Daftar semua aplikasi",
		"data":    apps,
		"meta": map[string]interface{}{
			"total":    total,
			"page":     page,
			"per_page": perPage,
		},
	})
}

// AdminGetPending — GET /v1/admin/apps/pending
func (h *AppHandler) AdminGetPending(c echo.Context) error {
	apps, err := h.appService.GetPending()
	if err != nil {
		return utils.InternalError(c, err.Error())
	}

	return utils.OK(c, "Aplikasi menunggu persetujuan", apps)
}

// AdminApprove — POST /v1/admin/apps/:id/approve
func (h *AppHandler) AdminApprove(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.BadRequest(c, "ID tidak valid")
	}

	if err := h.appService.Approve(id); err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.OK(c, "Aplikasi berhasil disetujui ✅", nil)
}

// AdminReject — POST /v1/admin/apps/:id/reject
func (h *AppHandler) AdminReject(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.BadRequest(c, "ID tidak valid")
	}

	if err := h.appService.Reject(id); err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.OK(c, "Aplikasi ditolak ❌", nil)
}

// AdminGetByID — GET /v1/admin/apps/:id
func (h *AppHandler) AdminGetByID(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.BadRequest(c, "ID tidak valid")
	}

	app, err := h.appService.AdminGetByID(id)
	if err != nil {
		return utils.NotFound(c, err.Error())
	}

	return utils.OK(c, "Detail aplikasi", app)
}

// AdminUpdate — PUT /v1/admin/apps/:id
func (h *AppHandler) AdminUpdate(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.BadRequest(c, "ID tidak valid")
	}

	var req service.UpdateAppRequest
	if err := c.Bind(&req); err != nil {
		return utils.BadRequest(c, "Request tidak valid")
	}

	app, err := h.appService.AdminUpdate(id, &req)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.OK(c, "Aplikasi berhasil diperbarui", app)
}

// AdminToggleActive — PUT /v1/admin/apps/:id/toggle-active
func (h *AppHandler) AdminToggleActive(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.BadRequest(c, "ID tidak valid")
	}

	app, err := h.appService.AdminToggleActive(id)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}

	status := "dinonaktifkan"
	if app.IsActive {
		status = "diaktifkan"
	}
	return utils.OK(c, "Aplikasi berhasil "+status, app)
}

// helper
func getClaims(c echo.Context) *utils.JWTClaims {
	claims, ok := c.Get(middleware.UserContextKey).(*utils.JWTClaims)
	if !ok {
		return nil
	}
	return claims
}
