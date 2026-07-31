package wechatoauth

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// stubTransport serves a canned response for any outbound request, so handler
// tests never touch the real WeChat API.
type stubTransport struct {
	status int
	body   string
}

func (s stubTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: s.status,
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Header:     make(http.Header),
	}, nil
}

// doRequest posts a JSON body to the given route and returns the response plus
// the error captured by a custom ErrorHandler.
func doRequest(t *testing.T, cfg Config, path string, body string) (*http.Response, error) {
	t.Helper()

	var captured error
	cfg.ErrorHandler = func(c fiber.Ctx, err error) error {
		captured = err
		return c.SendStatus(http.StatusBadRequest)
	}

	app := fiber.New()
	Register(app, cfg)

	req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	return resp, captured
}

func doAuthorizeLink(t *testing.T, cfg Config, body string) (*http.Response, error) {
	t.Helper()
	return doRequest(t, cfg, "/wechat/authorize-link", body)
}

func doOpenIDCallback(t *testing.T, cfg Config, body string) (*http.Response, error) {
	t.Helper()
	return doRequest(t, cfg, "/wechat/open-id-callback", body)
}

// decodeData unmarshals the {code,data,msg} envelope and returns data.
func decodeData(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	var envelope struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
		Msg  string         `json:"msg"`
	}
	if err = json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("unmarshal response %q: %v", raw, err)
	}
	return envelope.Data
}

func TestAuthorizeLinkGen_AllowedOrigin(t *testing.T) {
	cfg := Config{
		AppID:        "wxapp",
		AppSecret:    "secret",
		AllowOrigins: []string{"https://allowed.example.com"},
	}
	resp, captured := doAuthorizeLink(t, cfg,
		`{"redirect_uri":"https://allowed.example.com/cb","state":"s1"}`)

	if captured != nil {
		t.Fatalf("unexpected error for allowed origin: %v", captured)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestAuthorizeLinkGen_DisallowedOrigin(t *testing.T) {
	cfg := Config{
		AppID:        "wxapp",
		AppSecret:    "secret",
		AllowOrigins: []string{"https://allowed.example.com"},
	}
	_, captured := doAuthorizeLink(t, cfg,
		`{"redirect_uri":"https://evil.example.com/cb","state":"s1"}`)

	if captured != ErrRedirectURIMismatch {
		t.Errorf("captured error = %v, want ErrRedirectURIMismatch", captured)
	}
}

func TestAuthorizeLinkGen_EmptyOriginEntryDoesNotWhitelistAll(t *testing.T) {
	cfg := Config{
		AppID:        "wxapp",
		AppSecret:    "secret",
		AllowOrigins: []string{""},
	}
	_, captured := doAuthorizeLink(t, cfg,
		`{"redirect_uri":"https://evil.example.com/cb","state":"s1"}`)

	if captured != ErrRedirectURIMismatch {
		t.Errorf("captured error = %v, want ErrRedirectURIMismatch", captured)
	}
}

func TestAuthorizeLinkGen_MissingRedirectURI(t *testing.T) {
	cfg := Config{
		AppID:        "wxapp",
		AppSecret:    "secret",
		AllowOrigins: []string{"https://allowed.example.com"},
	}
	_, captured := doAuthorizeLink(t, cfg, `{"state":"s1"}`)

	if captured != ErrMissingRedirectURI {
		t.Errorf("captured error = %v, want ErrMissingRedirectURI", captured)
	}
}

func TestAuthorizeLinkGen_EscapesState(t *testing.T) {
	cfg := Config{
		AppID:        "wxapp",
		AppSecret:    "secret",
		AllowOrigins: []string{"https://allowed.example.com"},
	}
	resp, captured := doAuthorizeLink(t, cfg,
		`{"redirect_uri":"https://allowed.example.com/cb","state":"a&b"}`)

	if captured != nil {
		t.Fatalf("unexpected error: %v", captured)
	}
	link, _ := decodeData(t, resp)["url"].(string)
	if !strings.Contains(link, "state=a%26b") {
		t.Errorf("state not escaped in %q", link)
	}
}

func TestOpenIDCallback_Success(t *testing.T) {
	cfg := Config{
		AppID:     "wxapp",
		AppSecret: "secret",
		HTTPClient: &http.Client{Transport: stubTransport{
			status: http.StatusOK,
			body:   `{"access_token":"t","expires_in":7200,"openid":"o-123","scope":"snsapi_base"}`,
		}},
	}
	resp, captured := doOpenIDCallback(t, cfg, `{"code":"wx-code","state":"s1"}`)

	if captured != nil {
		t.Fatalf("unexpected error: %v", captured)
	}
	if openID, _ := decodeData(t, resp)["openID"].(string); openID != "o-123" {
		t.Errorf("openID = %q, want o-123", openID)
	}
}

func TestOpenIDCallback_UpstreamErrcode(t *testing.T) {
	cfg := Config{
		AppID:     "wxapp",
		AppSecret: "secret",
		HTTPClient: &http.Client{Transport: stubTransport{
			status: http.StatusOK,
			body:   `{"errcode":40029,"errmsg":"invalid code"}`,
		}},
	}
	_, captured := doOpenIDCallback(t, cfg, `{"code":"bad-code","state":"s1"}`)

	var ue *UpstreamError
	if !errors.As(captured, &ue) {
		t.Fatalf("captured error = %v, want *UpstreamError", captured)
	}
	if ue.ErrCode != 40029 {
		t.Errorf("ErrCode = %d, want 40029", ue.ErrCode)
	}
}

func TestOpenIDCallback_UpstreamHTTPStatus(t *testing.T) {
	cfg := Config{
		AppID:     "wxapp",
		AppSecret: "secret",
		HTTPClient: &http.Client{Transport: stubTransport{
			status: http.StatusBadGateway,
			body:   `<html>bad gateway</html>`,
		}},
	}
	_, captured := doOpenIDCallback(t, cfg, `{"code":"wx-code","state":"s1"}`)

	var ue *UpstreamError
	if !errors.As(captured, &ue) {
		t.Fatalf("captured error = %v, want *UpstreamError", captured)
	}
	if ue.HTTPStatus != http.StatusBadGateway {
		t.Errorf("HTTPStatus = %d, want 502", ue.HTTPStatus)
	}
}

func TestOpenIDCallback_MissingCode(t *testing.T) {
	cfg := Config{AppID: "wxapp", AppSecret: "secret"}
	_, captured := doOpenIDCallback(t, cfg, `{"state":"s1"}`)

	if captured != ErrMissingCode {
		t.Errorf("captured error = %v, want ErrMissingCode", captured)
	}
}

func TestOpenIDCallback_EmptyOpenID(t *testing.T) {
	cfg := Config{
		AppID:     "wxapp",
		AppSecret: "secret",
		HTTPClient: &http.Client{Transport: stubTransport{
			status: http.StatusOK,
			body:   `{"access_token":"t","expires_in":7200,"scope":"snsapi_base"}`,
		}},
	}
	_, captured := doOpenIDCallback(t, cfg, `{"code":"wx-code","state":"s1"}`)

	if captured != ErrEmptyOpenID {
		t.Errorf("captured error = %v, want ErrEmptyOpenID", captured)
	}
}

func TestNew_FillsDefaults(t *testing.T) {
	h := New(Config{AppID: "x"})
	if h.Config.ErrorHandler == nil {
		t.Error("ErrorHandler default not set")
	}
	if h.Config.Unmarshal == nil {
		t.Error("Unmarshal default not set")
	}
	if h.Config.HTTPClient == nil {
		t.Error("HTTPClient default not set")
	}
	if h.Config.HTTPClient.Timeout != defaultHTTPTimeout {
		t.Errorf("HTTPClient timeout = %v, want %v", h.Config.HTTPClient.Timeout, defaultHTTPTimeout)
	}
}
