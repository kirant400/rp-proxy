// handlers/mapping.go
package handlers

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// StationToArea mapping: station_id → area_id
var StationToArea = make(map[string]string)
var AreaToStations = make(map[string][]string)

// UploadMapping handles /mapping/:filename
func UploadMapping(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "filename required in path"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "form field 'file' is required"})
		return
	}
	defer file.Close()

	// Security: prevent path traversal
	if strings.ContainsAny(filename, "\\/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid characters in filename"})
		return
	}

	// Validate uploaded filename matches route
	if header.Filename != filename {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":     "filename mismatch",
			"expected":  filename,
			"uploaded":  header.Filename,
		})
		return
	}

	// Save file for audit
	uploadDir := "./uploads"
	os.MkdirAll(uploadDir, os.ModePerm)
	savePath := filepath.Join(uploadDir, filename)

	out, err := os.Create(savePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save file"})
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write file"})
		return
	}

	// Reset pointer for parsing
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot reread file"})
		return
	}

	// Parse based on extension
	ext := strings.ToLower(filepath.Ext(filename))
	var records int
	switch ext {
	case ".csv":
		records, err = parseCSVStationArea(file)
	case ".json":
		records, err = parseJSONStationArea(file)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "only .csv and .json supported"})
		return
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parsing failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        "station → area mapping loaded",
		"filename":      filename,
		"records":       records,
		"unique_areas":  len(AreaToStations),
		"saved_to":      savePath,
	})
}

// parseCSVStationArea parses CSV: station_id,area_id
func parseCSVStationArea(file multipart.File) (int, error) {
	reader := csv.NewReader(bufio.NewReader(file))
	reader.TrimLeadingSpace = true

	StationToArea = make(map[string]string)
	AreaToStations = make(map[string][]string)

	count := 0
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return count, err
		}
		if len(row) < 2 {
			continue
		}

		stationID := strings.TrimSpace(row[0])
		areaID := strings.TrimSpace(row[1])

		if stationID == "" || areaID == "" {
			continue
		}

		StationToArea[stationID] = areaID
		AreaToStations[areaID] = append(AreaToStations[areaID], stationID)
		count++
	}
	return count, nil
}

// parseJSONStationArea supports both object array and array-of-arrays
func parseJSONStationArea(file multipart.File) (int, error) {
	// Clear maps
	StationToArea = make(map[string]string)
	AreaToStations = make(map[string][]string)

	dec := json.NewDecoder(file)

	// Try array of objects first
	var entries []struct {
		StationID string `json:"station_id"`
		AreaID    string `json:"area_id"`
	}
	if err := dec.Decode(&entries); err == nil {
		count := 0
		for _, e := range entries {
			if e.StationID != "" && e.AreaID != "" {
				StationToArea[e.StationID] = e.AreaID
				AreaToStations[e.AreaID] = append(AreaToStations[e.AreaID], e.StationID)
				count++
			}
		}
		return count, nil
	}

	// Fallback: array of arrays [["ST001","AREA01"], ...]
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}
	dec = json.NewDecoder(file)

	var rows [][]string
	if err := dec.Decode(&rows); err != nil {
		return 0, err
	}

	count := 0
	for _, row := range rows {
		if len(row) >= 2 {
			stationID := strings.TrimSpace(row[0])
			areaID := strings.TrimSpace(row[1])
			if stationID != "" && areaID != "" {
				StationToArea[stationID] = areaID
				AreaToStations[areaID] = append(AreaToStations[areaID], stationID)
				count++
			}
		}
	}
	return count, nil
}