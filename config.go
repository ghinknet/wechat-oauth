package wechatoauth

import (
	"net/http"

	"github.com/gofiber/fiber/v3"
)

// Config configures the WeChat web OAuth (snsapi_base) helper.
type Config struct {
	// AppID is the WeChat official account / open platform app id.
	AppID string
	// AppSecret is the corresponding app secret.
	AppSecret string
	// AllowOrigins whitelists redirect_uri prefixes for AuthorizeLink. Empty
	// entries are ignored, so a blank origin can never whitelist everything.
	AllowOrigins []string

	// ErrorHandler handles errors raised inside the handlers. When nil, a
	// default 500 JSON response is used.
	ErrorHandler func(c fiber.Ctx, err error) error
	// Unmarshal customises JSON decoding. Defaults to encoding/json.
	Unmarshal func(data []byte, v any) error
	// HTTPClient performs the outbound WeChat API calls. When nil, a default
	// client with a 10-second timeout is used.
	HTTPClient *http.Client
}
