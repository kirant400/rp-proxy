// handlers/reset.go
package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kirant400/rp-wrapper/cache"
)

// Use the global context from cache package (defined in redis.go)
var redisCtx = context.Background()  // ← This is the correct way in go-redis/v9

// ResetMapping – clears in-memory station → area mapping
func ResetMapping(c *gin.Context) {
	StationToArea = make(map[string]string)
	AreaToStations = make(map[string][]string)

	c.JSON(http.StatusOK, gin.H{
		"status":  "mapping reset",
		"message": "station → area mapping cleared",
	})
}

// ResetDepartments – deletes all department keys from Redis
func ResetDepartments(c *gin.Context) {
	keys, err := cache.Client.Keys(redisCtx, "dept*").Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Redis KEYS failed", "details": err.Error()})
		return
	}
	if len(keys) > 0 {
		cache.Client.Del(redisCtx, keys...)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":             "departments reset",
		"redis_keys_deleted": len(keys),
	})
}

// ResetAreas
func ResetAreas(c *gin.Context) {
	keys, _ := cache.Client.Keys(redisCtx, "area*").Result()
	if len(keys) > 0 {
		cache.Client.Del(redisCtx, keys...)
	}
	c.JSON(http.StatusOK, gin.H{"status": "areas reset", "deleted_keys": len(keys)})
}

// ResetPositions
func ResetPositions(c *gin.Context) {
	keys, _ := cache.Client.Keys(redisCtx, "pos*").Result()
	if len(keys) > 0 {
		cache.Client.Del(redisCtx, keys...)
	}
	c.JSON(http.StatusOK, gin.H{"status": "positions reset", "deleted_keys": len(keys)})
}

// ResetEmployees
func ResetEmployees(c *gin.Context) {
	keys, _ := cache.Client.Keys(redisCtx, "emp*").Result()
	if len(keys) > 0 {
		cache.Client.Del(redisCtx, keys...)
	}
	c.JSON(http.StatusOK, gin.H{"status": "employees reset", "deleted_keys": len(keys)})
}

// ResetAllAttendance – clears in-memory attendance flags
func ResetAllAttendance(c *gin.Context) {
	Attendance = make(map[string]bool)
	c.JSON(http.StatusOK, gin.H{"status": "all attendance reset"})
}

// Nuclear option – wipes everything
func ResetEverything(c *gin.Context) {
	cache.Client.FlushDB(redisCtx)
	StationToArea = make(map[string]string)
	AreaToStations = make(map[string][]string)
	Attendance = make(map[string]bool)

	c.JSON(http.StatusOK, gin.H{
		"status":  "full reset completed",
		"message": "Redis DB flushed + all in-memory state cleared",
	})
}