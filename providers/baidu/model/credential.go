package model

const (
	DefaultClientID     = "hq9yQ9w9kR4YHj1kyYafLygVocobh7Sf"
	DefaultClientSecret = "YH2VpZcFJHYNnV6vLfHQXDBhcE7ZChyE"
)

type Credential struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
}

func (c Credential) WithDefaults() Credential {
	if c.ClientID == "" {
		c.ClientID = DefaultClientID
	}
	if c.ClientSecret == "" {
		c.ClientSecret = DefaultClientSecret
	}
	return c
}

func (c Credential) RedactedRefreshToken() string {
	token := c.RefreshToken
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "***" + token[len(token)-4:]
}
