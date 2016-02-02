package uaa_go_client

type Token struct {
	AccessToken string `json:"access_token"`
	// Expire time in seconds
	ExpiresIn int64 `json:"expires_in"`
}
