package handler

import (
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"sso.pelajarnumagetan.or.id/internal/service"
	"sso.pelajarnumagetan.or.id/internal/utils"
)

type UserHandler struct {
	userService service.UserService
}

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// AdminGetUsers — GET /v1/admin/users
func (h *UserHandler) AdminGetUsers(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if perPage < 1 {
		perPage = 10
	}

	users, total, err := h.userService.GetAll(page, perPage)
	if err != nil {
		return utils.InternalError(c, err.Error())
	}

	// Hilangkan password dari response
	type userResp struct {
		ID         uuid.UUID `json:"id"`
		Name       string    `json:"name"`
		Email      string    `json:"email"`
		Phone      string    `json:"phone"`
		Image      string    `json:"image"`
		Role       string    `json:"role"`
		IsActive   bool      `json:"is_active"`
		IsVerified bool      `json:"is_verified"`
	}

	result := make([]userResp, len(users))
	for i, u := range users {
		result[i] = userResp{
			ID:         u.ID,
			Name:       u.Name,
			Email:      u.Email,
			Phone:      u.Phone,
			Image:      u.Image,
			Role:       string(u.Role),
			IsActive:   u.IsActive,
			IsVerified: u.IsVerified,
		}
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"message": "Daftar user",
		"data":    result,
		"meta": map[string]interface{}{
			"total":    total,
			"page":     page,
			"per_page": perPage,
		},
	})
}

// AdminUpdateRole — PUT /v1/admin/users/:id/role
func (h *UserHandler) AdminUpdateRole(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.BadRequest(c, "ID tidak valid")
	}

	type roleReq struct {
		Role string `json:"role"`
	}
	var req roleReq
	if err := c.Bind(&req); err != nil || req.Role == "" {
		return utils.BadRequest(c, "role diperlukan")
	}

	if err := h.userService.UpdateRole(id, req.Role); err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.OK(c, "Role user berhasil diubah menjadi: "+req.Role, nil)
}

// AdminDeactivate — PUT /v1/admin/users/:id/deactivate
func (h *UserHandler) AdminDeactivate(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.BadRequest(c, "ID tidak valid")
	}

	if err := h.userService.Deactivate(id); err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.OK(c, "User berhasil dinonaktifkan", nil)
}

// AdminGetByID — GET /v1/admin/users/:id
func (h *UserHandler) AdminGetByID(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.BadRequest(c, "ID tidak valid")
	}

	user, err := h.userService.GetByID(id)
	if err != nil {
		return utils.NotFound(c, err.Error())
	}

	return utils.OK(c, "Detail user", map[string]interface{}{
		"id":         user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"phone":      user.Phone,
		"image":      user.Image,
		"role":       string(user.Role),
		"is_active":  user.IsActive,
		"is_verified": user.IsVerified,
		"created_at": user.CreatedAt,
	})
}

// AdminActivate — PUT /v1/admin/users/:id/activate
func (h *UserHandler) AdminActivate(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.BadRequest(c, "ID tidak valid")
	}

	if err := h.userService.Activate(id); err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.OK(c, "User berhasil diaktifkan", nil)
}

// AdminVerifyEmail — PUT /v1/admin/users/:id/verify-email
func (h *UserHandler) AdminVerifyEmail(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.BadRequest(c, "ID tidak valid")
	}

	if err := h.userService.VerifyEmail(id); err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.OK(c, "Email user berhasil diverifikasi secara manual", nil)
}

// AdminDelete — DELETE /v1/admin/users/:id
func (h *UserHandler) AdminDelete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return utils.BadRequest(c, "ID tidak valid")
	}

	if err := h.userService.Delete(id); err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.OK(c, "User berhasil dihapus", nil)
}
