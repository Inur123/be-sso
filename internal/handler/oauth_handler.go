package handler

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"sso.pelajarnumagetan.or.id/internal/middleware"
	"sso.pelajarnumagetan.or.id/internal/service"
	"sso.pelajarnumagetan.or.id/internal/utils"
)

type OAuthHandler struct {
	oauthService service.OAuthService
}

func NewOAuthHandler(oauthService service.OAuthService) *OAuthHandler {
	return &OAuthHandler{oauthService: oauthService}
}

// Authorize — GET /oauth/authorize
// Dipanggil FE saat user diarahkan dari App A ke halaman consent SSO
// FE perlu kirim Bearer token user yang sedang login
func (h *OAuthHandler) Authorize(c echo.Context) error {
	req := &service.AuthorizeRequest{
		ResponseType: c.QueryParam("response_type"),
		ClientID:     c.QueryParam("client_id"),
		RedirectURI:  c.QueryParam("redirect_uri"),
		Scope:        c.QueryParam("scope"),
		State:        c.QueryParam("state"),
	}

	info, err := h.oauthService.Authorize(req)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}

	// Kembalikan info app ke FE untuk tampilkan halaman consent
	return utils.OK(c, "Informasi consent", info)
}

// Confirm — POST /oauth/authorize/confirm
// Dipanggil FE setelah user klik "Izinkan" — butuh Bearer token
func (h *OAuthHandler) Confirm(c echo.Context) error {
	claims, ok := c.Get(middleware.UserContextKey).(*utils.JWTClaims)
	if !ok || claims == nil {
		return utils.Unauthorized(c, "Silakan login terlebih dahulu")
	}

	req := &service.AuthorizeRequest{}
	if err := c.Bind(req); err != nil {
		return utils.BadRequest(c, "Request tidak valid")
	}

	code, err := h.oauthService.Confirm(claims.UserID, req)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}

	// Kembalikan redirect URL ke FE, FE yang redirect browser-nya
	redirectURL := fmt.Sprintf("%s?code=%s&state=%s", req.RedirectURI, code, req.State)
	return utils.OK(c, "Authorization code berhasil dibuat", map[string]string{
		"redirect_url": redirectURL,
		"code":         code,
	})
}

// Token — POST /oauth/token
// Dipanggil backend App A untuk tukar code → access_token
func (h *OAuthHandler) Token(c echo.Context) error {
	var req service.TokenRequest
	if err := c.Bind(&req); err != nil {
		return utils.BadRequest(c, "Request tidak valid")
	}

	// Dukung Basic Auth header (standart OAuth2 RFC 6749) jika client credentials kosong di JSON body
	if clientID, clientSecret, ok := c.Request().BasicAuth(); ok {
		if req.ClientID == "" {
			req.ClientID = clientID
		}
		if req.ClientSecret == "" {
			req.ClientSecret = clientSecret
		}
	}

	resp, err := h.oauthService.ExchangeToken(&req)
	if err != nil {
		return utils.Unauthorized(c, err.Error())
	}

	// Standar OAuth2 Token Response: data token HARUS berada di root JSON, bukan di dalam objek "data"
	return c.JSON(200, resp)
}

// RefreshAccessToken — POST /oauth/refreshAccessToken
// Dipanggil App A saat access_token expired
func (h *OAuthHandler) RefreshAccessToken(c echo.Context) error {
	var req service.RefreshRequest
	if err := c.Bind(&req); err != nil {
		return utils.BadRequest(c, "Request tidak valid")
	}

	// Dukung Basic Auth header jika client credentials kosong di JSON body
	if clientID, _, ok := c.Request().BasicAuth(); ok {
		if req.ClientID == "" {
			req.ClientID = clientID
		}
	}

	resp, err := h.oauthService.RefreshAccessToken(&req)
	if err != nil {
		return utils.Unauthorized(c, err.Error())
	}

	// Standar OAuth2: token response di root JSON
	return c.JSON(200, resp)
}

// Revoke — POST /oauth/revoke
// Dipanggil App A saat user logout dari App A
func (h *OAuthHandler) Revoke(c echo.Context) error {
	type revokeReq struct {
		Token string `json:"token"`
	}

	var req revokeReq
	if err := c.Bind(&req); err != nil || req.Token == "" {
		return utils.BadRequest(c, "token diperlukan")
	}

	if err := h.oauthService.RevokeToken(req.Token); err != nil {
		return utils.BadRequest(c, err.Error())
	}

	// Sesuai nu.id — revoke tidak ada response body
	return c.NoContent(200)
}
