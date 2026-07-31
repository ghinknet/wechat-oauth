package wechatoauth

// AccessTokenResponse is the WeChat sns oauth2 access_token response.
type AccessTokenResponse struct {
	AccessToken    string `json:"access_token"`
	ExpiresIn      int    `json:"expires_in"`
	RefreshToken   string `json:"refresh_token"`
	OpenID         string `json:"openid"`
	Scope          string `json:"scope"`
	IsSnapshotUser int    `json:"is_snapshotuser"`
	UnionID        string `json:"unionid"`
	// ErrCode / ErrMsg carry WeChat's business failure; WeChat reports errors
	// in the body with HTTP 200, so 0 is the only success value.
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// OpenIDCallbackRequest binds the code/state posted back from the front-end.
type OpenIDCallbackRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

// AuthorizeLinkRequest binds an authorise-link generation request.
type AuthorizeLinkRequest struct {
	RedirectURI string `json:"redirect_uri" binding:"required"`
	State       string `json:"state" binding:"required"`
}
