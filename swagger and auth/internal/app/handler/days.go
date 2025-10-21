package handler

import (
    "net/http"
    "regexp"
    "strconv"
    "strings"
    "time"
    "front_start/internal/app/repository"
    "front_start/internal/app/auth"

    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
    minio "front_start/internal/app/minio"
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
        "asteroidRequestId": obsID,
        "timestamp":    time.Now().Unix(),
    })
}


// API: Days

// ApiListDays godoc
// @Summary      List days
// @Description  Get list of astronomy days with optional name filter
// @Tags         Days
// @Produce      json
// @Param        name query string false "Filter by day name"
// @Success      200 {object} map[string]interface{}
// @Failure      500 {object} map[string]string
// @Router       /api/days [get]
func (h *Handler) ApiListDays(c *gin.Context) {
    name := strings.TrimSpace(c.Query("name"))
    if name == "" {
        days, err := h.Repository.GetDays()
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
            return
        }
        c.JSON(http.StatusOK, gin.H{"items": days})
        return
    }
    days, err := h.Repository.GetDaysByDate(name)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"items": days})
}

// ApiGetDayAPI godoc
// @Summary      Get day by ID
// @Description  Get astronomy day details by ID
// @Tags         Days
// @Produce      json
// @Param        id path int true "Day ID"
// @Success      200 {object} repository.Day
// @Failure      400 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /api/days/{id} [get]
func (h *Handler) ApiGetDayAPI(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }
    d, err := h.Repository.GetDay(id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
        return
    }
    c.JSON(http.StatusOK, d)
}

type serviceCreateDTO struct {
    Name        string  `json:"name" binding:"required"`
    Description string  `json:"description"`
    BodiesText  string  `json:"bodiesText"`
    EarthRA     float64 `json:"earthRA"`
    EarthDEC    float64 `json:"earthDEC"`
}


// ApiCreateDay godoc
// @Summary      Create new day
// @Description  Create new astronomy day (moderator only)
// @Tags         Days
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body serviceCreateDTO true "Day data"
// @Success      201 {object} map[string]int
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /api/days [post]
func (h *Handler) ApiCreateDay(c *gin.Context) {
    if bad, key, err := checkForbiddenJSONKeys(c, []string{"id", "is_deleted", "image_url"}); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"}); return
    } else if bad {
        c.JSON(http.StatusForbidden, gin.H{"error": "forbidden field", "field": key}); return
    }
    var in serviceCreateDTO
    if err := c.ShouldBindJSON(&in); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
        return
    }
    id, err := h.Repository.CreateService(in.Name, in.Description, in.BodiesText, in.EarthRA, in.EarthDEC)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
        return
    }
    c.JSON(http.StatusCreated, gin.H{"id": id})
}

type serviceUpdateDTO struct {
    Name        *string  `json:"name"`
    Description *string  `json:"description"`
    BodiesText  *string  `json:"bodiesText"`
    EarthRA     *float64 `json:"earthRA"`
    EarthDEC    *float64 `json:"earthDEC"`
}

// ApiUpdateDay godoc
// @Summary      Update day
// @Description  Update astronomy day (moderator only)
// @Tags         Days
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path int true "Day ID"
// @Param        request body serviceUpdateDTO true "Update data"
// @Success      204
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      404 {object} map[string]string
// @Router       /api/days/{id} [put]
func (h *Handler) ApiUpdateDay(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }
    if bad, key, err := checkForbiddenJSONKeys(c, []string{"id", "is_deleted", "image_url"}); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"}); return
    } else if bad {
        c.JSON(http.StatusForbidden, gin.H{"error": "forbidden field", "field": key}); return
    }
    var in serviceUpdateDTO
    if err := c.ShouldBindJSON(&in); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
        return
    }
    if err := h.Repository.UpdateService(id, in.Name, in.Description, in.BodiesText, in.EarthRA, in.EarthDEC); err != nil {
        status := http.StatusInternalServerError
        if err != nil {
            status = http.StatusNotFound
        }
        c.JSON(status, gin.H{"error": "update failed"})
        return
    }
    c.Status(http.StatusNoContent)
}

// ApiDeleteDay godoc
// @Summary      Delete day
// @Description  Soft delete astronomy day (moderator only)
// @Tags         Days
// @Security     BearerAuth
// @Produce      json
// @Param        id path int true "Day ID"
// @Success      204
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /api/days/{id} [delete]
func (h *Handler) ApiDeleteDay(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }
    oldURL, _ := h.Repository.GetServiceImageURL(id)
    if oldURL != "" {
        if mc, err := minio.New(); err == nil {
            _ = mc.DeleteByURL(c, oldURL)
        }
    }
    if err := h.Repository.SoftDeleteService(id); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
        return
    }
    c.Status(http.StatusNoContent)
}

// POST /api/days/:id/add-to-draft
// ApiDayAddToDraft godoc
// @Summary      Add day to draft
// @Description  Add day to current user's draft asteroid request
// @Tags         Days
// @Security     BearerAuth
// @Produce      json
// @Param        id path int true "Day ID"
// @Success      204
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /api/days/{id}/add-to-draft [post]
func (h *Handler) ApiDayAddToDraft(c *gin.Context) {
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
    
    if _, err := h.Repository.AddServiceToDraft(int(user.ID), id); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
        return
    }
    c.Status(http.StatusNoContent)
}

// ApiDayUploadImage godoc
// @Summary      Upload day image
// @Description  Upload image for astronomy day (moderator only)
// @Tags         Days
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        id path int true "Day ID"
// @Param        file formData file true "Image file"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      403 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /api/days/{id}/image [post]
func (h *Handler) ApiDayUploadImage(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }
    file, err := c.FormFile("file")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
        return
    }
    safeName := regexp.MustCompile(`[^a-zA-Z0-9._-]`).ReplaceAllString(file.Filename, "_")
    objectName := strconv.Itoa(id) + "_" + safeName
    oldURL, _ := h.Repository.GetServiceImageURL(id)
    f, err := file.Open()
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "file open failed"})
        return
    }
    defer f.Close()
    minioClient, err := minio.New()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "storage init failed"})
        return
    }
    uploadedURL, err := minioClient.Upload(c, objectName, f, file.Size, file.Header.Get("Content-Type"))
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed"})
        return
    }
    if err := h.Repository.UpdateServiceImageURL(id, uploadedURL); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
        return
    }
    _ = minioClient.DeleteByURL(c, oldURL)
    c.JSON(http.StatusOK, gin.H{"imageURL": uploadedURL})
}

