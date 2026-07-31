package wechatoauth

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
)

// defaultHTTPTimeout bounds outbound WeChat API calls so a hanging upstream
// cannot block a handler forever.
const defaultHTTPTimeout = 10 * time.Second

// Handler holds the OAuth handlers bound to a Config.
type Handler struct {
	Config Config
}

// New builds a Handler, filling in defaults for optional Config fields.
func New(config Config) *Handler {
	if config.ErrorHandler == nil {
		config.ErrorHandler = respInternalServerError
	}
	if config.Unmarshal == nil {
		config.Unmarshal = json.Unmarshal
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: defaultHTTPTimeout}
	}
	return &Handler{Config: config}
}

// OpenIDCallback exchanges a code for the user's OpenID via WeChat sns oauth2.
func (h *Handler) OpenIDCallback(c fiber.Ctx) error {
	// Read request params
	var req OpenIDCallbackRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.Config.ErrorHandler(c, err)
	}
	if req.Code == "" {
		return h.Config.ErrorHandler(c, ErrMissingCode)
	}

	// Build request URI; the code is caller-supplied, escape it so it cannot
	// inject extra query parameters.
	requestURL := strings.Join([]string{
		"https://api.weixin.qq.com/sns/oauth2/access_token?appid=",
		h.Config.AppID,
		"&secret=",
		h.Config.AppSecret,
		"&code=",
		url.QueryEscape(req.Code),
		"&grant_type=authorization_code",
	}, "")

	// Send request, bounded by the inbound request context
	httpReq, err := http.NewRequestWithContext(c.Context(), http.MethodGet, requestURL, nil)
	if err != nil {
		return h.Config.ErrorHandler(c, err)
	}
	resp, err := h.Config.HTTPClient.Do(httpReq)
	if err != nil {
		return h.Config.ErrorHandler(c, err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return h.Config.ErrorHandler(c, err)
	}
	if resp.StatusCode != http.StatusOK {
		return h.Config.ErrorHandler(c, &UpstreamError{HTTPStatus: resp.StatusCode})
	}

	// Parse JSON data
	var result AccessTokenResponse
	if err = h.Config.Unmarshal(body, &result); err != nil {
		return h.Config.ErrorHandler(c, err)
	}

	// WeChat reports failures inside a 200 body: errcode != 0 means the code
	// was invalid/expired/used, never a valid-but-empty openid.
	if result.ErrCode != 0 {
		return h.Config.ErrorHandler(c, &UpstreamError{ErrCode: result.ErrCode, ErrMsg: result.ErrMsg})
	}
	if result.OpenID == "" {
		return h.Config.ErrorHandler(c, ErrEmptyOpenID)
	}

	return respSuccess(c, fiber.Map{"openID": result.OpenID})
}

// AuthorizeLinkGen builds a WeChat web authorise link (snsapi_base scope).
func (h *Handler) AuthorizeLinkGen(c fiber.Ctx) error {
	// Read request params
	var req AuthorizeLinkRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.Config.ErrorHandler(c, err)
	}
	if req.RedirectURI == "" {
		return h.Config.ErrorHandler(c, ErrMissingRedirectURI)
	}

	// Check allowed origins
	allowed := false
	for _, origin := range h.Config.AllowOrigins {
		if origin == "" {
			continue
		}
		if strings.HasPrefix(req.RedirectURI, origin) {
			allowed = true
			break
		}
	}
	if !allowed {
		return h.Config.ErrorHandler(c, ErrRedirectURIMismatch)
	}

	// Encode caller-supplied values so they cannot break the authorise URL
	redirectURI := url.QueryEscape(req.RedirectURI)
	state := url.QueryEscape(req.State)

	authURL := strings.Join([]string{
		"https://open.weixin.qq.com/connect/oauth2/authorize?appid=",
		h.Config.AppID,
		"&redirect_uri=",
		redirectURI,
		"&response_type=code&scope=snsapi_base&state=",
		state,
		"#wechat_redirect",
	}, "")

	return respSuccess(c, fiber.Map{"url": authURL})
}
