package handler

import (
    "net/http"
    "regexp"
    "strconv"
    "strings"
    "time"
    "front_start/internal/app/repository"

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
        "observationId": obsID,
        "timestamp":    time.Now().Unix(),
    })
}


// API: Days

// GET /api/days?name=...
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

// GET /api/days/:id
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

// POST /api/days (без изображения)
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

// PUT /api/days/:id
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

// DELETE /api/days/:id
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
func (h *Handler) ApiDayAddToDraft(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
        return
    }
    userID := 1
    if _, err := h.Repository.AddServiceToDraft(userID, id); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
        return
    }
    c.Status(http.StatusNoContent)
}

// POST /api/days/:id/image
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

