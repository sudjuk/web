package handler

import (
    "front_start/internal/app/repository"
    "net/http"
    "strconv"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
)

type Handler struct {
	Repository *repository.Repository
}

func NewHandler(r *repository.Repository) *Handler {
	return &Handler{
		Repository: r,
	}
}

func (h *Handler) GetDays(ctx *gin.Context) {
	var days []repository.Day
	var err error
	observationId := 1

	searchQuery := ctx.Query("name") // получаем значение из поля поиска
	if searchQuery == "" {            // если поле поиска пусто, то просто получаем из репозитория все записи
		days, err = h.Repository.GetDays()
    if err != nil {
        logrus.Error(err)
        ctx.String(http.StatusInternalServerError, "internal error")
        return
    }
	} else {
		days, err = h.Repository.GetDaysByDate(searchQuery) // в ином случае ищем день по дате
        if err != nil {
            logrus.Error(err)
            ctx.String(http.StatusInternalServerError, "internal error")
            return
        }
	}

    obsDays, err := h.Repository.GetObservationDays(observationId)
    if err != nil {
        logrus.Error(err)
        ctx.String(http.StatusInternalServerError, "internal error")
        return
    }
    currentCounter := len(obsDays)

	ctx.HTML(http.StatusOK, "days_list.html", gin.H{
		"days":     days,
		"query":    searchQuery,
		"counter":  currentCounter,
		"observationId": observationId,
		"timestamp": time.Now().Unix(),
	})
}

func (h *Handler) GetDay(ctx *gin.Context) {
	idStr := ctx.Param("id") // получаем id дня из урла (то есть из /day/:id)
	// через двоеточие мы указываем параметры, которые потом сможем считать через функцию выше
	id, err := strconv.Atoi(idStr) // так как функция выше возвращает нам строку, нужно ее преобразовать в int
        if err != nil {
            logrus.Error(err)
            ctx.String(http.StatusInternalServerError, "internal error")
            return
        }

    day, err := h.Repository.GetDay(id)
    if err != nil {
        if err == repository.ErrDayNotFound {
            ctx.String(http.StatusNotFound, "404. day not found")
            return
        }
        logrus.Error(err)
        ctx.String(http.StatusInternalServerError, "internal error")
        return
    }

	ctx.HTML(http.StatusOK, "day_details.html", gin.H{
		"day": day,
	})
}

func (h *Handler) GetObservation(ctx *gin.Context) {
	idStr := ctx.Param("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
        logrus.Error(err)
        ctx.String(http.StatusBadRequest, "invalid id")
        return
	}

    observation, err := h.Repository.GetObservation(id)
    if err != nil {
        logrus.Error(err)
        ctx.String(http.StatusInternalServerError, "internal error")
        return
    }

    // получаем список дней для заявки
    obsDays, err := h.Repository.GetObservationDays(id)
    if err != nil {
        logrus.Error(err)
        ctx.String(http.StatusInternalServerError, "internal error")
        return
    }

    ctx.HTML(http.StatusOK, "observation.html", gin.H{
        "observation":      observation,
        "observationDays":  obsDays,
        "result":          observation.Result,
    })
}
