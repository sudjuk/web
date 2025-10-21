package auth

import (
    "errors"
    "time"

    "front_start/internal/app/ds"
    "front_start/internal/app/repository"
    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v4"
)

const UserContextKey = "user"

// GenerateToken создает JWT токен для пользователя
func GenerateToken(user repository.PublicUser, jwtSecret string, expiresIn time.Duration) (string, error) {
    claims := &ds.JWTClaims{
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Issuer:    "astronomy-api",
        },
        UserID:      user.ID,
        Login:       user.Login,
        IsModerator: user.IsModerator,
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(jwtSecret))
}

// ParseToken парсит и валидирует JWT токен
func ParseToken(tokenString, jwtSecret string) (*ds.JWTClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &ds.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, errors.New("unexpected signing method")
        }
        return []byte(jwtSecret), nil
    })

    if err != nil {
        return nil, err
    }

    if claims, ok := token.Claims.(*ds.JWTClaims); ok && token.Valid {
        return claims, nil
    }

    return nil, errors.New("invalid token")
}

// GetUserFromContext извлекает пользователя из gin.Context
func GetUserFromContext(c *gin.Context) (repository.PublicUser, error) {
    userInterface, exists := c.Get(UserContextKey)
    if !exists {
        return repository.PublicUser{}, errors.New("user not found in context")
    }

    user, ok := userInterface.(repository.PublicUser)
    if !ok {
        return repository.PublicUser{}, errors.New("invalid user type in context")
    }

    return user, nil
}
