package handler

import (
    "errors"
    "net/http"
    "strings"

    "front_start/internal/app/auth"
    "front_start/internal/app/repository"
    "github.com/gin-gonic/gin"
    "github.com/go-redis/redis/v8"
)

const BearerPrefix = "Bearer "

// WithAuthCheck проверяет авторизацию через JWT или cookie сессию
func (h *Handler) WithAuthCheck() gin.HandlerFunc {
    return func(c *gin.Context) {
        var user repository.PublicUser
        var err error

        // Приоритет 1: JWT токен из заголовка Authorization
        authHeader := c.GetHeader("Authorization")
        if strings.HasPrefix(authHeader, BearerPrefix) {
            tokenString := authHeader[len(BearerPrefix):]
            user, err = h.authenticateByJWT(tokenString)
            if err == nil {
                c.Set(auth.UserContextKey, user)
                c.Next()
                return
            }
        }

        // Приоритет 2: Cookie сессия
        sessionID, err := c.Cookie("session_id")
        if err == nil && sessionID != "" {
            user, err = h.authenticateBySession(c, sessionID)
            if err == nil {
                c.Set(auth.UserContextKey, user)
                c.Next()
                return
            }
        }

        // Если ни один способ не сработал
        c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
    }
}

// WithModeratorCheck проверяет авторизацию и роль модератора
func (h *Handler) WithModeratorCheck() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Сначала проверяем авторизацию
        var user repository.PublicUser
        var err error

        // Приоритет 1: JWT токен из заголовка Authorization
        authHeader := c.GetHeader("Authorization")
        if strings.HasPrefix(authHeader, BearerPrefix) {
            tokenString := authHeader[len(BearerPrefix):]
            user, err = h.authenticateByJWT(tokenString)
            if err == nil {
                c.Set(auth.UserContextKey, user)
            }
        } else {
            // Если нет JWT токена, пробуем cookie
            sessionID, cookieErr := c.Cookie("session_id")
            if cookieErr == nil && sessionID != "" {
                user, err = h.authenticateBySession(c, sessionID)
                if err == nil {
                    c.Set(auth.UserContextKey, user)
                }
            } else {
                err = errors.New("no authentication method found")
            }
        }

        // Если авторизация не прошла
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
            return
        }

        // Проверяем роль модератора
        if !user.IsModerator {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "error": "moderator required",
                "debug": gin.H{
                    "user_id": user.ID,
                    "login": user.Login,
                    "is_moderator": user.IsModerator,
                },
            })
            return
        }

        c.Next()
    }
}

// authenticateByJWT аутентифицирует пользователя по JWT токену
func (h *Handler) authenticateByJWT(tokenString string) (repository.PublicUser, error) {
    claims, err := auth.ParseToken(tokenString, h.JWTSecret)
    if err != nil {
        return repository.PublicUser{}, err
    }

    // Загружаем актуальные данные пользователя из БД
    user, err := h.Repository.GetUserByID(claims.UserID)
    if err != nil {
        return repository.PublicUser{}, err
    }

    return user, nil
}

// authenticateBySession аутентифицирует пользователя по сессии в Redis
func (h *Handler) authenticateBySession(c *gin.Context, sessionID string) (repository.PublicUser, error) {
    userID, err := h.RedisClient.GetUserIDBySession(c.Request.Context(), sessionID)
    if err != nil {
        if errors.Is(err, redis.Nil) {
            return repository.PublicUser{}, errors.New("session not found")
        }
        return repository.PublicUser{}, err
    }

    // Загружаем данные пользователя из БД
    user, err := h.Repository.GetUserByID(userID)
    if err != nil {
        return repository.PublicUser{}, err
    }

    return user, nil
}
