package handler

import (
    "fmt"
    "net/http"
    "strconv"
    "time"

    "github.com/gin-gonic/gin"
)

type registerDTO struct { Login string `json:"login" binding:"required"`; Password string `json:"password" binding:"required"` }
type loginDTO struct { Login string `json:"login" binding:"required"`; Password string `json:"password" binding:"required"` }
type profileUpdateDTO struct { Login *string `json:"login"`; Password *string `json:"password"` }

func (h *Handler) ApiRegister(c *gin.Context) {
    var in registerDTO
    if err := c.ShouldBindJSON(&in); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"}); return }
    u, err := h.Repository.Register(in.Login, in.Password)
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
    c.JSON(http.StatusCreated, u)
}

func (h *Handler) ApiLogin(c *gin.Context) {
    var in loginDTO
    if err := c.ShouldBindJSON(&in); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"}); return }
    u, err := h.Repository.Authenticate(in.Login, in.Password)
    if err != nil { c.JSON(http.StatusUnauthorized, gin.H{"error": "bad credentials"}); return }
    // простая cookie-сессия
    c.SetCookie("uid", fmt.Sprintf("%d", u.ID), int((24*time.Hour).Seconds()), "/", "", false, true)
    c.JSON(http.StatusOK, u)
}

func (h *Handler) ApiLogout(c *gin.Context) {
    c.SetCookie("uid", "", -1, "/", "", false, true)
    c.Status(http.StatusNoContent)
}

func (h *Handler) ApiGetMe(c *gin.Context) {
    uid, err := c.Cookie("uid")
    if err != nil || uid == "" { c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"}); return }
    id, _ := strconv.ParseInt(uid, 10, 64)
    u, err := h.Repository.GetUserByID(id)
    if err != nil { c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"}); return }
    c.JSON(http.StatusOK, u)
}

func (h *Handler) ApiUpdateMe(c *gin.Context) {
    uid, err := c.Cookie("uid")
    if err != nil || uid == "" { c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"}); return }
    id, _ := strconv.ParseInt(uid, 10, 64)
    var in profileUpdateDTO
    if err := c.ShouldBindJSON(&in); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"}); return }
    if err := h.Repository.UpdateProfile(id, in.Login, in.Password); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "update failed"}); return }
    c.Status(http.StatusNoContent)
}


