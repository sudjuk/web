package handler

import (
    "net/http"
    "time"

    "front_start/internal/app/auth"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

type registerDTO struct { 
    Login    string `json:"login" binding:"required"` 
    Password string `json:"password" binding:"required"` 
}

type loginDTO struct { 
    Login    string `json:"login" binding:"required"` 
    Password string `json:"password" binding:"required"` 
}

type loginResponseDTO struct {
    AccessToken string `json:"access_token"`
    TokenType   string `json:"token_type"`
    ExpiresIn   int64  `json:"expires_in"` // в секундах
}

type profileUpdateDTO struct { 
    Login    *string `json:"login"` 
    Password *string `json:"password"` 
}

// ApiRegister godoc
// @Summary      Register new user
// @Description  Create a new user account
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body registerDTO true "Registration data"
// @Success      201 {object} repository.PublicUser
// @Failure      400 {object} map[string]string
// @Router       /api/auth/register [post]
func (h *Handler) ApiRegister(c *gin.Context) {
    var in registerDTO
    if err := c.ShouldBindJSON(&in); err != nil { 
        c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
        return 
    }
    u, err := h.Repository.Register(in.Login, in.Password)
    if err != nil { 
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return 
    }
    c.JSON(http.StatusCreated, u)
}

// ApiLogin godoc
// @Summary      Login user
// @Description  Authenticate user and return JWT token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body loginDTO true "Login credentials"
// @Success      200 {object} loginResponseDTO
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Router       /api/auth/login [post]
func (h *Handler) ApiLogin(c *gin.Context) {
    var in loginDTO
    if err := c.ShouldBindJSON(&in); err != nil { 
        c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
        return 
    }
    
    u, err := h.Repository.Authenticate(in.Login, in.Password)
    if err != nil { 
        c.JSON(http.StatusUnauthorized, gin.H{"error": "bad credentials", "details": err.Error()})
        return 
    }

    // Генерируем sessionID для cookie
    sessionID := uuid.New().String()
    
    // Сохраняем сессию в Redis (24 часа)
    ttl := 24 * time.Hour
    err = h.RedisClient.SaveSession(c.Request.Context(), sessionID, u.ID, ttl)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "session creation failed"})
        return
    }

    // Устанавливаем cookie для браузера
    c.SetCookie("session_id", sessionID, int(ttl.Seconds()), "/", "", false, true)

    // Генерируем JWT токен для API клиентов
    jwtToken, err := auth.GenerateToken(u, h.JWTSecret, ttl)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
        return
    }

    c.JSON(http.StatusOK, loginResponseDTO{
        AccessToken: jwtToken,
        TokenType:   "Bearer",
        ExpiresIn:   int64(ttl.Seconds()),
    })
}

// ApiLogout godoc
// @Summary      Logout user
// @Description  Invalidate user session
// @Tags         Auth
// @Security     BearerAuth
// @Success      204
// @Failure      401 {object} map[string]string
// @Router       /api/auth/logout [post]
func (h *Handler) ApiLogout(c *gin.Context) {
    // Получаем sessionID из cookie
    sessionID, err := c.Cookie("session_id")
    if err == nil && sessionID != "" {
        // Удаляем сессию из Redis
        h.RedisClient.DeleteSession(c.Request.Context(), sessionID)
    }
    
    // Очищаем cookie
    c.SetCookie("session_id", "", -1, "/", "", false, true)
    c.Status(http.StatusNoContent)
}

// ApiGetMe godoc
// @Summary      Get current user
// @Description  Get information about currently authenticated user
// @Tags         Profile
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} repository.PublicUser
// @Failure      401 {object} map[string]string
// @Router       /api/me [get]
func (h *Handler) ApiGetMe(c *gin.Context) {
    user, err := auth.GetUserFromContext(c)
    if err != nil { 
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return 
    }
    c.JSON(http.StatusOK, user)
}

// ApiUpdateMe godoc
// @Summary      Update current user profile
// @Description  Update current user's login and/or password
// @Tags         Profile
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body profileUpdateDTO true "Profile update data"
// @Success      204
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Router       /api/me [put]
func (h *Handler) ApiUpdateMe(c *gin.Context) {
    user, err := auth.GetUserFromContext(c)
    if err != nil { 
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return 
    }
    
    var in profileUpdateDTO
    if err := c.ShouldBindJSON(&in); err != nil { 
        c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
        return 
    }
    
    if err := h.Repository.UpdateProfile(user.ID, in.Login, in.Password); err != nil { 
        c.JSON(http.StatusBadRequest, gin.H{"error": "update failed"})
        return 
    }
    c.Status(http.StatusNoContent)
}


