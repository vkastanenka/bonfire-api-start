package auth

type RefreshParams struct {
	RefreshToken string
}

type RefreshResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RefreshRes struct {
	AccessToken string `json:"access_token"`
}
