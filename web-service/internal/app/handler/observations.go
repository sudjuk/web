package handler

import (
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
    auth "front_start/internal/app/auth"
)

func (h *Handler) GetDay(ctx *gin.Context) {
    idStr := ctx.Param("id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        logrus.Error(err)
        ctx.String(http.StatusInternalServerError, "internal error")
        return
    }
    day, err := h.Repository.GetDay(id)
    if err != nil {
        if err != nil {
            ctx.String(http.StatusNotFound, "404. day not found")
            return
        }
        logrus.Error(err)
        ctx.String(http.StatusInternalServerError, "internal error")
        return
    }
    ctx.HTML(http.StatusOK, "day_details.html", gin.H{"day": day})
}

func (h *Handler) GetObservation(ctx *gin.Context) {
    idStr := ctx.Param("id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        logrus.Error(err)
        ctx.String(http.StatusBadRequest, "invalid id")
        return
    }
    deleted, err := h.Repository.IsObservationDeleted(id)
    if err != nil {
        logrus.Error(err)
        ctx.String(http.StatusInternalServerError, "internal error")
        return
    }
    if deleted {
        ctx.String(http.StatusNotFound, "404. observation not found")
        return
    }
    observation, err := h.Repository.GetObservation(id)
    if err != nil {
        logrus.Error(err)
        ctx.String(http.StatusInternalServerError, "internal error")
        return
    }
    obsDays, err := h.Repository.GetObservationDays(id)
    if err != nil {
        logrus.Error(err)
        ctx.String(http.StatusInternalServerError, "internal error")
        return
    }
    ctx.HTML(http.StatusOK, "observation.html", gin.H{
        "observation":     observation,
        "observationDays": obsDays,
        "result":          observation.Result,
    })
}

// PostAddToOrder adds a day to current user's draft observation via ORM
func (h *Handler) PostAddToOrder(ctx *gin.Context) {
    userID := 1
    dayIDStr := ctx.PostForm("day_id")
    dayID, err := strconv.Atoi(dayIDStr)
    if err != nil || dayID <= 0 {
        ctx.String(http.StatusBadRequest, "invalid day_id")
        return
    }
    _, err = h.Repository.AddServiceToDraft(userID, dayID)
    if err != nil {
        logrus.Error(err)
        ctx.String(http.StatusInternalServerError, "internal error")
        return
    }
    referer := ctx.Request.Referer()
    if referer == "" {
        referer = "/astronomy"
    }
    ctx.Redirect(http.StatusSeeOther, referer)
}

// PostDeleteOrder marks observation deleted via raw SQL
func (h *Handler) PostDeleteOrder(ctx *gin.Context) {
    userID := 1
    obsIDStr := ctx.PostForm("observation_id")
    obsID, err := strconv.Atoi(obsIDStr)
    if err != nil || obsID <= 0 {
        ctx.String(http.StatusBadRequest, "invalid observation_id")
        return
    }
    if err := h.Repository.SoftDeleteObservationRaw(userID, obsID); err != nil {
        logrus.Error(err)
        ctx.String(http.StatusInternalServerError, "internal error")
        return
    }
    ctx.Redirect(http.StatusSeeOther, "/astronomy")
}


// API: Observations

func (h *Handler) ApiGetCartIcon(c *gin.Context) {
    // По требованию лабы: создатель фиксирован константой (singleton)
    userID := auth.CurrentUserID()
    obsID, cnt, err := h.Repository.CountDraftItems(userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"draftId": obsID, "count": cnt})
}

func (h *Handler) ApiListObservations(c *gin.Context) {
    status := c.Query("status")
    var statusPtr *string
    if strings.TrimSpace(status) != "" { s := strings.TrimSpace(status); statusPtr = &s }
    var fromPtr, toPtr *time.Time
    if v := strings.TrimSpace(c.Query("from")); v != "" { if t, err := time.Parse("2006-01-02", v); err == nil { fromPtr = &t } }
    if v := strings.TrimSpace(c.Query("to")); v != "" { if t, err := time.Parse("2006-01-02", v); err == nil { toPtr = &t } }
    rows, err := h.Repository.ListObservations(statusPtr, fromPtr, toPtr)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"}); return }
    c.JSON(http.StatusOK, gin.H{"items": rows})
}

func (h *Handler) ApiGetObservation(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
    deleted, err := h.Repository.IsObservationDeleted(id)
    if err != nil || deleted { c.JSON(http.StatusNotFound, gin.H{"error": "not found"}); return }
    obs, err := h.Repository.GetObservation(id)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"}); return }
    days, err := h.Repository.GetObservationDays(id)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"}); return }
    c.JSON(http.StatusOK, gin.H{"observation": obs, "items": days})
}

type observationUpdateDTO struct { Comment *string `json:"comment"` }

func (h *Handler) ApiUpdateObservation(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
    if bad, key, err := checkForbiddenJSONKeys(c, []string{"id", "status", "creator_id", "moderator_id", "created_at", "formed_at", "completed_at"}); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"}); return
    } else if bad { c.JSON(http.StatusForbidden, gin.H{"error": "forbidden field", "field": key}); return }
    var in observationUpdateDTO
    if err := c.ShouldBindJSON(&in); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"}); return }
    userID := auth.CurrentUserID()
    if err := h.Repository.UpdateObservation(userID, id, in.Comment); err != nil { c.JSON(http.StatusNotFound, gin.H{"error": "update failed"}); return }
    c.Status(http.StatusNoContent)
}

func (h *Handler) ApiSubmitObservation(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
    userID := auth.CurrentUserID()
    if err := h.Repository.SubmitObservation(userID, id); err != nil { logrus.Error(err); c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state or not found"}); return }
    c.Status(http.StatusNoContent)
}

type moderateDTO struct { Action string `json:"action" binding:"required"` }

func (h *Handler) ApiModerateObservation(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
    var in moderateDTO
    if err := c.ShouldBindJSON(&in); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"}); return }
    act := strings.ToLower(strings.TrimSpace(in.Action))
    if act != "complete" && act != "reject" { c.JSON(http.StatusBadRequest, gin.H{"error": "action must be complete or reject"}); return }
    uid := currentUserIDFromCookie64(c)
    u, err := h.Repository.GetUserByID(uid)
    if err != nil || !u.IsModerator { c.JSON(http.StatusForbidden, gin.H{"error": "moderator required"}); return }
    if err := h.Repository.CompleteOrReject(int(uid), id, act == "complete"); err != nil { logrus.Error(err); c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state or not found"}); return }
    c.Status(http.StatusNoContent)
}

// helpers: current user from cookie with fallback to constant=1 (требование лабы)
func currentUserIDFromCookie(c *gin.Context) int {
    if uid, err := c.Cookie("uid"); err == nil {
        if v, err := strconv.Atoi(uid); err == nil && v > 0 { return v }
    }
    return 1
}

func currentUserIDFromCookie64(c *gin.Context) int64 {
    if uid, err := c.Cookie("uid"); err == nil {
        if v, err := strconv.ParseInt(uid, 10, 64); err == nil && v > 0 { return v }
    }
    return 1
}

// m-m API inside observations

type mmUpdateDTO struct {
    ObservationID int      `json:"observationId" binding:"required"`
    DayID         int      `json:"dayId" binding:"required"`
    Quantity      *int     `json:"quantity"`
    SortOrder     *int     `json:"sortOrder"`
    IsPrimary     *bool    `json:"isPrimary"`
    Note          *string  `json:"note"`
    AsteroidRA    *float64 `json:"asteroidRA"`
    AsteroidDEC   *float64 `json:"asteroidDEC"`
}

func (h *Handler) ApiDeleteObservationItem(c *gin.Context) {
    obsID, _ := strconv.Atoi(c.Query("observationId"))
    dayID, _ := strconv.Atoi(c.Query("dayId"))
    if obsID <= 0 || dayID <= 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "observationId and dayId required"})
        return
    }
    userID := auth.CurrentUserID()
    if err := h.Repository.DeleteObservationItem(userID, obsID, dayID); err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "delete failed"})
        return
    }
    c.Status(http.StatusNoContent)
}

func (h *Handler) ApiUpdateObservationItem(c *gin.Context) {
    if bad, key, err := checkForbiddenJSONKeys(c, []string{"observation_id", "day_id"}); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"}); return
    } else if bad {
        c.JSON(http.StatusForbidden, gin.H{"error": "forbidden field", "field": key}); return
    }
    var in mmUpdateDTO
    if err := c.ShouldBindJSON(&in); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
        return
    }
    userID := auth.CurrentUserID()
    if err := h.Repository.UpdateObservationItem(userID, in.ObservationID, in.DayID, in.Quantity, in.SortOrder, in.IsPrimary, in.Note, in.AsteroidRA, in.AsteroidDEC); err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "update failed"})
        return
    }
    c.Status(http.StatusNoContent)
}

// DELETE /api/observations/:id — мягкое удаление заявки создателем
func (h *Handler) ApiDeleteObservation(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }
    userID := auth.CurrentUserID()
    if err := h.Repository.SoftDeleteObservationRaw(userID, id); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
        return
    }
    c.Status(http.StatusNoContent)
}
