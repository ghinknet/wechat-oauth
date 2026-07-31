# wechat-oauth

A small, standalone helper for the **WeChat web OAuth** (`snsapi_base`) flow used
to obtain a user's **OpenID**. GoFiber only.

## Module

```
go.gh.ink/wechat-oauth
```

```bash
go get go.gh.ink/wechat-oauth
```

## What it does

The WeChat web OAuth flow has two steps; this library exposes one endpoint for
each:

1. **Generate an authorise link** — your front-end redirects the user to it;
   WeChat redirects back to your `redirect_uri` with a `code`.
2. **Exchange the code for an OpenID** — your front-end posts the `code` back and
   receives the `openID`.

## Usage

Mount the routes on a GoFiber router with `Register`:

```go
import (
    "github.com/gofiber/fiber/v3"

    wechatoauth "go.gh.ink/wechat-oauth"
)

app := fiber.New()

wechatoauth.Register(app, wechatoauth.Config{
    AppID:        "wx-app-id",
    AppSecret:    "wx-app-secret",
    AllowOrigins: []string{"https://app.example.com"}, // redirect_uri whitelist
})
```

`Register` mounts a `/wechat` group on the given router, so relative to it:

| Method & path                 | Purpose |
|-------------------------------|---------|
| `POST /wechat/authorize-link` | Build a WeChat authorize link. |
| `POST /wechat/open-id-callback` | Exchange a `code` for an `openID`. |

### `POST /wechat/authorize-link`

Request:

```json
{ "redirect_uri": "https://app.example.com/cb", "state": "csrf-token" }
```

`redirect_uri` must start with one of `Config.AllowOrigins`, otherwise the
request is rejected with `ErrRedirectURIMismatch`.

Response:

```json
{ "code": 200, "data": { "url": "https://open.weixin.qq.com/connect/oauth2/authorize?..." }, "msg": "success" }
```

### `POST /wechat/open-id-callback`

Request (the `code` WeChat appended to your `redirect_uri`):

```json
{ "code": "wx-code", "state": "csrf-token" }
```

Response:

```json
{ "code": 200, "data": { "openID": "o-xxxx" }, "msg": "success" }
```

When WeChat rejects the code (invalid / expired / already used), the handler
does **not** return an empty `openID`: the failure is routed to `ErrorHandler`
as an `*UpstreamError` carrying WeChat's `errcode` / `errmsg`.

## Configuration (`Config`)

| Field          | Meaning |
|----------------|---------|
| `AppID`        | WeChat official-account / open-platform app ID. |
| `AppSecret`    | Corresponding app secret. |
| `AllowOrigins` | Allowed `redirect_uri` prefixes for `authorizeLink`. Empty entries are ignored. |
| `ErrorHandler` | Optional `func(c fiber.Ctx, err error) error`. Defaults to a 500 JSON response. |
| `Unmarshal`    | Optional custom JSON decoder. Defaults to `encoding/json`. |
| `HTTPClient`   | Optional `*http.Client` for WeChat API calls. Defaults to a client with a 10s timeout. Outbound calls are also bound to the inbound request context. |

## Errors

Handlers report failures through `ErrorHandler`:

- `ErrRedirectURIMismatch` — `redirect_uri` not covered by `AllowOrigins`.
- `ErrMissingRedirectURI` / `ErrMissingCode` — required request field absent.
- `ErrEmptyOpenID` — WeChat answered success without an `openid`.
- `*UpstreamError` — WeChat API failure; match with `errors.As` and inspect
  `HTTPStatus` (non-200 transport reply) or `ErrCode` / `ErrMsg` (business
  failure inside a 200 body).

## Advanced

`Register` is a convenience wrapper. For full control over routing, build a
`*Handler` yourself and mount the handlers where you like:

```go
h := wechatoauth.New(cfg)
app.Post("/oauth/link", h.AuthorizeLinkGen)
app.Post("/oauth/openid", h.OpenIDCallback)
```

## License

See [LICENSE](LICENSE).
