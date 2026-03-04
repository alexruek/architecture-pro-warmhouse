package main

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Device struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Location     string    `json:"location"`
	Status       string    `json:"status"`
	HouseID      string    `json:"house_id"`
	RegisteredAt time.Time `json:"registered_at"`
}

type RegisterDeviceRequest struct {
	Name     string `json:"name"     binding:"required"`
	Type     string `json:"type"     binding:"required"`
	Location string `json:"location" binding:"required"`
	HouseID  string `json:"house_id" binding:"required"`
}

var (
	devices = make(map[string]Device)
	mu      sync.RWMutex
)

func main() {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "device-service"})
	})

	r.GET("/devices", func(c *gin.Context) {
		mu.RLock()
		defer mu.RUnlock()
		list := make([]Device, 0, len(devices))
		for _, d := range devices {
			list = append(list, d)
		}
		c.JSON(http.StatusOK, list)
	})

	r.GET("/devices/:id", func(c *gin.Context) {
		mu.RLock()
		defer mu.RUnlock()
		d, ok := devices[c.Param("id")]
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
			return
		}
		c.JSON(http.StatusOK, d)
	})

	r.POST("/devices", func(c *gin.Context) {
		var req RegisterDeviceRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		device := Device{
			ID:           uuid.New().String(),
			Name:         req.Name,
			Type:         req.Type,
			Location:     req.Location,
			Status:       "OFFLINE",
			HouseID:      req.HouseID,
			RegisteredAt: time.Now(),
		}
		mu.Lock()
		devices[device.ID] = device
		mu.Unlock()
		c.JSON(http.StatusCreated, device)
	})

	r.PATCH("/devices/:id/status", func(c *gin.Context) {
		var body struct {
			Status string `json:"status" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		mu.Lock()
		defer mu.Unlock()
		d, ok := devices[c.Param("id")]
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
			return
		}
		d.Status = body.Status
		devices[d.ID] = d
		c.JSON(http.StatusOK, d)
	})

	r.DELETE("/devices/:id", func(c *gin.Context) {
		mu.Lock()
		defer mu.Unlock()
		if _, ok := devices[c.Param("id")]; !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
			return
		}
		delete(devices, c.Param("id"))
		c.JSON(http.StatusNoContent, nil)
	})

	log.Println("Device Service starting on :8082")
	r.Run(":8082")
}
