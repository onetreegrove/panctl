package model

const (
	DriveTypeDefault  = "default"
	DriveTypeResource = "resource"
	DriveTypeBackup   = "backup"
)

type Credential struct {
	RefreshToken  string `json:"refresh_token"`
	AccessToken   string `json:"access_token,omitempty"`
	ClientID      string `json:"client_id,omitempty"`
	ClientSecret  string `json:"client_secret,omitempty"`
	OAuthTokenURL string `json:"oauth_token_url,omitempty"`
	DriveType     string `json:"drive_type,omitempty"`
	DriveID       string `json:"drive_id,omitempty"`
	UserID        string `json:"user_id,omitempty"`
}

func (c Credential) WithDefaults() Credential {
	switch c.DriveType {
	case DriveTypeDefault, DriveTypeResource, DriveTypeBackup:
	default:
		// The Alist open driver defaults to the resource drive; keep CLI behavior aligned.
		c.DriveType = DriveTypeResource
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
