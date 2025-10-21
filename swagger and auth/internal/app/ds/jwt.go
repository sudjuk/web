package ds

import (
    "github.com/golang-jwt/jwt/v4"
)

type JWTClaims struct {
    jwt.RegisteredClaims
    UserID      int64  `json:"user_id"`
    Login       string `json:"login"`
    IsModerator bool   `json:"is_moderator"`
}
