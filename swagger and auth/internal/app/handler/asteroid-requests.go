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

func (h *Handler) GetAsteroidObservation(ctx *gin.Context) {
    idStr := ctx.Param("id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        logrus.Error(err)
        ctx.String(http.StatusBadRequest, "invalid id")
        return
    }
    deleted, err := h.Repository.IsAsteroidObservationDeleted(id)
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
    obsDays, err := h.Repository.GetAsteroidObservationDays(id)
    if err != nil {
        logrus.Error(err)
        ctx.String(http.StatusInternalServerError, "internal error")
        return
    }
    ctx.HTML(http.StatusOK, "asteroid-observation.html", gin.H{
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
    if err := h.Repository.SoftDeleteAsteroidObservationRaw(userID, obsID); err != nil {
        logrus.Error(err)
        ctx.String(http.StatusInternalServerError, "internal error")
        return
    }
    ctx.Redirect(http.StatusSeeOther, "/astronomy")
}


// API: Observations

// ApiGetCartIcon godoc
// @Summary      Get cart icon info
// @Description  Get draft observation ID and item count for current user
// @Tags         Asteroid Observations
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /api/asteroid-observations/cart [get]
func (h *Handler) ApiGetCartIcon(c *gin.Context) {
    user, err := auth.GetUserFromContext(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }
    
    obsID, cnt, err := h.Repository.CountDraftItems(int(user.ID))
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"draftId": obsID, "count": cnt})
}

// ApiListAsteroidObservations godoc
// @Summary      List asteroid observations
// @Description  Get list of asteroid observations with optional filtering
// @Tags         Asteroid Observations
// @Produce      json
// @Param        status query string false "Filter by status"
// @Param        from query string false "Filter from date (YYYY-MM-DD)"
// @Param        to query string false "Filter to date (YYYY-MM-DD)"
// @Success      200 {object} map[string]interface{}
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /api/asteroid-observations [get]
func (h *Handler) ApiListAsteroidObservations(c *gin.Context) {
    // Получаем пользователя из контекста (middleware уже проверил авторизацию)
    user, err := auth.GetUserFromContext(c)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
        return
    }

    status := c.Query("status")
    var statusPtr *string
    if strings.TrimSpace(status) != "" { 
        s := strings.TrimSpace(status)
        statusPtr = &s 
    }
    
    var fromPtr, toPtr *time.Time
    if v := strings.TrimSpace(c.Query("from")); v != "" { 
        if t, err := time.Parse("2006-01-02", v); err == nil { 
            fromPtr = &t 
        } 
    }
    if v := strings.TrimSpace(c.Query("to")); v != "" { 
        if t, err := time.Parse("2006-01-02", v); err == nil { 
            toPtr = &t 
        } 
    }

    // Определяем фильтр по создателю
    var creatorID *int64
    if !user.IsModerator {
        // Обычный пользователь видит только свои заявки
        creatorID = &user.ID
    }
    // Модератор видит все заявки (creatorID остается nil)
    
    // Отладочная информация
    logrus.Infof("User ID: %d, IsModerator: %v, CreatorID filter: %v", user.ID, user.IsModerator, creatorID)

    rows, err := h.Repository.ListAsteroidObservations(statusPtr, fromPtr, toPtr, creatorID)
    if err != nil { 
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
        return 
    }
    c.JSON(http.StatusOK, gin.H{"items": rows})
}

// ApiGetAsteroidRequest godoc
// @Summary      Get asteroid observation by ID
// @Description  Get asteroid observation details by ID
// @Tags         Asteroid Observations
// @Produce      json
// @Param        id path int true "Asteroid request ID"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /api/asteroid-observations/{id} [get]
func (h *Handler) ApiGetAsteroidObservation(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }
    deleted, err := h.Repository.IsAsteroidObservationDeleted(id)
    if err != nil || deleted { c.JSON(http.StatusNotFound, gin.H{"error": "not found"}); return }
    obs, err := h.Repository.GetObservation(id)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"}); return }
    days, err := h.Repository.GetAsteroidObservationDays(id)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"}); return }
    c.JSON(http.StatusOK, gin.H{"observation": obs, "items": days})
}

type asteroidRequestUpdateDTO struct { Comment *string `json:"comment"` }

// ApiUpdateAsteroidObservation godoc
// @Summary      Update asteroid observation
// @Description  Update asteroid observation comment (only by creator)
// @Tags         Asteroid Observations
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path int true "Asteroid request ID"
// @Param        request body asteroidRequestUpdateDTO true "Update data"
// @Success      204
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /api/asteroid-observations/{id} [put]
func (h *Handler) ApiUpdateAsteroidObservation(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 { 
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return 
    }
    
    if bad, key, err := checkForbiddenJSONKeys(c, []string{"id", "status", "creator_id", "moderator_id", "created_at", "formed_at", "completed_at"}); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
        return
    } else if bad { 
        c.JSON(http.StatusForbidden, gin.H{"error": "forbidden field", "field": key})
        return 
    }
    
	var in asteroidRequestUpdateDTO
    if err := c.ShouldBindJSON(&in); err != nil { 
        c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
        return 
    }
    
    user, err := auth.GetUserFromContext(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }
    
    if err := h.Repository.UpdateAsteroidObservation(int(user.ID), id, in.Comment); err != nil { 
        c.JSON(http.StatusNotFound, gin.H{"error": "update failed"})
        return 
    }
    c.Status(http.StatusNoContent)
}

// ApiSubmitAsteroidObservation godoc
// @Summary      Submit asteroid observation
// @Description  Submit draft asteroid observation for moderation
// @Tags         Asteroid Observations
// @Security     BearerAuth
// @Produce      json
// @Param        id path int true "Asteroid request ID"
// @Success      204
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Router       /api/asteroid-observations/{id}/submit [put]
func (h *Handler) ApiSubmitAsteroidObservation(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 { 
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return 
    }
    
    user, err := auth.GetUserFromContext(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }
    
    if err := h.Repository.SubmitAsteroidObservation(int(user.ID), id); err != nil { 
        logrus.Error(err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state or not found"})
        return 
    }
    c.Status(http.StatusNoContent)
}

type moderateDTO struct { 
    Action string `json:"action" binding:"required"` 
    Comment *string `json:"comment"`
}

// ApiModerateAsteroidRequest godoc
// @Summary      Moderate asteroid observation
// @Description  Complete or reject asteroid observation (moderator only)
// @Tags         Asteroid Observations
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path int true "Asteroid request ID"
// @Param        request body moderateDTO true "Moderation action"
// @Success      204
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Router       /api/asteroid-observations/{id}/moderate [put]
func (h *Handler) ApiModerateAsteroidObservation(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 { 
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return 
    }
    
    var in moderateDTO
    if err := c.ShouldBindJSON(&in); err != nil { 
        c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
        return 
    }
    
    act := strings.ToLower(strings.TrimSpace(in.Action))
    if act != "complete" && act != "reject" { 
        c.JSON(http.StatusBadRequest, gin.H{"error": "action must be complete or reject"})
        return 
    }
    
    user, err := auth.GetUserFromContext(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }
    
    if !user.IsModerator { 
        c.JSON(http.StatusForbidden, gin.H{"error": "moderator required"})
        return 
    }
    
    if err := h.Repository.CompleteOrReject(int(user.ID), id, act == "complete", in.Comment); err != nil { 
        logrus.Error(err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state or not found"})
        return 
    }
    c.Status(http.StatusNoContent)
}


// m-m API inside observations

type mmUpdateDTO struct {
    ObservationID int      `json:"asteroidRequestId" binding:"required"`
    DayID         int      `json:"dayId" binding:"required"`
    AsteroidRA    *float64 `json:"asteroidRA"`
    AsteroidDEC   *float64 `json:"asteroidDEC"`
}

// ApiDeleteAsteroidObservationItem godoc
// @Summary      Delete asteroid observation item
// @Description  Delete item from asteroid observation (only by creator)
// @Tags         Asteroid Observations
// @Security     BearerAuth
// @Produce      json
// @Param        asteroidRequestId query int true "Asteroid request ID"
// @Param        dayId query int true "Day ID"
// @Success      204
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /api/asteroid-request-items [delete]
func (h *Handler) ApiDeleteAsteroidObservationItem(c *gin.Context) {
    obsID, _ := strconv.Atoi(c.Query("asteroidRequestId"))
    dayID, _ := strconv.Atoi(c.Query("dayId"))
    if obsID <= 0 || dayID <= 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "asteroidRequestId and dayId required"})
        return
    }
    
    user, err := auth.GetUserFromContext(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }
    
    if err := h.Repository.DeleteAsteroidObservationItem(int(user.ID), obsID, dayID); err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "delete failed"})
        return
    }
    c.Status(http.StatusNoContent)
}

// ApiUpdateAsteroidObservationItem godoc
// @Summary      Update asteroid observation item
// @Description  Update asteroid coordinates in request item (only by creator)
// @Tags         Asteroid Observations
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body mmUpdateDTO true "Update data"
// @Success      204
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /api/asteroid-request-items [put]
func (h *Handler) ApiUpdateAsteroidObservationItem(c *gin.Context) {
    if bad, key, err := checkForbiddenJSONKeys(c, []string{"observation_id", "day_id"}); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
        return
    } else if bad {
        c.JSON(http.StatusForbidden, gin.H{"error": "forbidden field", "field": key})
        return
    }
    
    var in mmUpdateDTO
    if err := c.ShouldBindJSON(&in); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
        return
    }
    
    user, err := auth.GetUserFromContext(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }
    
    if err := h.Repository.UpdateAsteroidObservationItem(int(user.ID), in.ObservationID, in.DayID, in.AsteroidRA, in.AsteroidDEC); err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "update failed"})
        return
    }
    c.Status(http.StatusNoContent)
}

// ApiDeleteAsteroidRequest godoc
// @Summary      Delete asteroid observation
// @Description  Soft delete asteroid observation (only by creator)
// @Tags         Asteroid Observations
// @Security     BearerAuth
// @Produce      json
// @Param        id path int true "Asteroid request ID"
// @Success      204
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /api/asteroid-observations/{id} [delete]
func (h *Handler) ApiDeleteAsteroidObservation(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }
    
    user, err := auth.GetUserFromContext(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }
    
    if err := h.Repository.SoftDeleteAsteroidObservationRaw(int(user.ID), id); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
        return
    }
    c.Status(http.StatusNoContent)
}
