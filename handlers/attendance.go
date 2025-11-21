// handlers/attendance.go
package handlers

import (
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

// Attendance map: station_id -> present (true/false)
var Attendance = make(map[string]bool)

// StationToEmployee mapping (populated when uploading mapping file)
var StationMapping = make(map[string]string) // station_id -> employee_name_or_id
var EmployeeToStation = make(map[string]string) // employee -> station_id (reverse lookup)

// Get current attendance status for a single station
func GetAttendance(c *gin.Context) {
	stationID := c.Param("stationid")

	present := Attendance[stationID]
	employee := StationMapping[stationID]

	c.JSON(http.StatusOK, gin.H{
		"station_id": stationID,
		"employee":   employee,
		"present":     present,
		"timestamp":   time.Now().Unix(),
	})
}

// Set attendance (mark as present)
func SetAttendance(c *gin.Context) {
	stationID := c.Param("stationid")
	Attendance[stationID] = true

	employee := StationMapping[stationID]

	c.JSON(http.StatusOK, gin.H{
		"status":      "present",
		"station_id":  stationID,
		"employee":    employee,
		"present":     true,
		"timestamp":   time.Now().Unix(),
	})
}

// Reset attendance for a specific station
func ResetAttendance(c *gin.Context) {
	stationID := c.Param("stationid")
	delete(Attendance, stationID)

	c.JSON(http.StatusOK, gin.H{
		"status":     "reset",
		"station_id": stationID,
	})
}

// === NEW: Display full attendance mapping ===
func DisplayAttendanceMapping(c *gin.Context) {
	// Collect all stations from mapping
	var mapping []gin.H

	// Sort station IDs for consistent output
	var stationIDs []string
	for stationID := range StationMapping {
		stationIDs = append(stationIDs, stationID)
	}
	sort.Strings(stationIDs)

	for _, stationID := range stationIDs {
		employee := StationMapping[stationID]
		present := Attendance[stationID]

		mapping = append(mapping, gin.H{
			"station_id": stationID,
			"employee":   employee,
			"present":    present,
			"last_seen":  func() int64 {
				if present {
					return time.Now().Unix()
				}
				return 0
			}(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"total_stations": len(mapping),
		"present_count":  countPresent(),
		"mapping":        mapping,
		"generated_at":   time.Now().Format(time.RFC3339),
	})
}

func countPresent() int {
	count := 0
	for _, present := range Attendance {
		if present {
			count++
		}
	}
	return count
}