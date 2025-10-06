package handler

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
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


