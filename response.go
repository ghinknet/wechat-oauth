package wechatoauth

import (
	"net/http"

	"github.com/gofiber/fiber/v3"
)

type response[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

func respSuccess[T any](c fiber.Ctx, data T) error {
	return c.Status(http.StatusOK).JSON(response[T]{
		Code: http.StatusOK,
		Msg:  "success",
		Data: data,
	})
}

func respInternalServerError(c fiber.Ctx, _ error) error {
	return c.Status(http.StatusInternalServerError).JSON(response[any]{
		Code: http.StatusInternalServerError,
		Msg:  "internal server error",
		Data: nil,
	})
}
