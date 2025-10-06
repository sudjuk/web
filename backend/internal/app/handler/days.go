package handler

import (
    "net/http"
    "time"
    "front_start/internal/app/repository"

    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
)

// GetDays: list and search services (days)
func (h *Handler) GetDays(ctx *gin.Context) {
    var err error
    userID := 1
    obsID, counter, err := h.Repository.CountDraftItems(userID)
    if err != nil {
        logrus.Error(err)
        ctx.String(http.StatusInternalServerError, "internal error")
        return
    }

    searchQuery := ctx.Query("name")
    var days []repository.Day
    if searchQuery == "" {
        d, err := h.Repository.GetDays()
        if err != nil {
            logrus.Error(err)
            ctx.String(http.StatusInternalServerError, "internal error")
            return
        }
        days = d
    } else {
        d, err := h.Repository.GetDaysByDate(searchQuery)
        if err != nil {
            logrus.Error(err)
            ctx.String(http.StatusInternalServerError, "internal error")
            return
        }
        days = d
    }

    ctx.HTML(http.StatusOK, "days_list.html", gin.H{
        "days":         days,
        "query":        searchQuery,
        "counter":      counter,
        "observationId": obsID,
        "timestamp":    time.Now().Unix(),
    })
}


