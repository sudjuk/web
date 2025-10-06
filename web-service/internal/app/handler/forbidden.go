package handler

import (
    "bytes"
    "encoding/json"
    "io"

    "github.com/gin-gonic/gin"
)

// checkForbiddenJSONKeys reads request body once, checks for forbidden keys, and restores body
func checkForbiddenJSONKeys(c *gin.Context, forbidden []string) (bool, string, error) {
    data, err := io.ReadAll(c.Request.Body)
    if err != nil {
        return false, "", err
    }
    // restore body for next bind
    c.Request.Body = io.NopCloser(bytes.NewBuffer(data))
    if len(data) == 0 {
        return false, "", nil
    }
    var m map[string]any
    if err := json.Unmarshal(data, &m); err != nil {
        return false, "", err
    }
    for _, k := range forbidden {
        if _, ok := m[k]; ok {
            return true, k, nil
        }
    }
    return false, "", nil
}






