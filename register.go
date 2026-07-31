package wechatoauth

import "github.com/gofiber/fiber/v3"

// Register mounts the WeChat OAuth routes on the given GoFiber router.
//
//	POST {prefix}/openIDCallback  -> exchange code for OpenID
//	POST {prefix}/authorizeLink   -> generate an authorise link
func Register(router fiber.Router, config Config) *Handler {
	h := New(config)
	group := router.Group("/wechat")
	group.Post("/open-id-callback", h.OpenIDCallback)
	group.Post("/authorize-link", h.AuthorizeLinkGen)
	return h
}
