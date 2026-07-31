package wechatoauth

import (
	"errors"
	"fmt"
)

var ErrRedirectURIMismatch = errors.New("wechat redirect uri mismatch")
var ErrMissingCode = errors.New("wechat oauth code is required")
var ErrMissingRedirectURI = errors.New("redirect uri is required")
var ErrEmptyOpenID = errors.New("wechat returned an empty openid")

// UpstreamError reports a failed exchange with the WeChat API: either a
// non-200 HTTP status, or a business failure carried in the response body
// (errcode != 0). Match it with errors.As to inspect the detail.
type UpstreamError struct {
	HTTPStatus int
	ErrCode    int
	ErrMsg     string
}

func (e *UpstreamError) Error() string {
	if e.ErrCode != 0 {
		return fmt.Sprintf("wechat upstream error: errcode=%d errmsg=%q", e.ErrCode, e.ErrMsg)
	}
	return fmt.Sprintf("wechat upstream error: http status %d", e.HTTPStatus)
}
