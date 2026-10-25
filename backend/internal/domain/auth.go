package domain

import "time"

type Session struct {
	AccessToken           string
	AccessTokenExpiresAt  time.Time
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
	User                  User
}

type AdminCredentials struct {
	User                       User
	TemporaryPassword          string
	TemporaryPasswordExpiresAt time.Time
}
