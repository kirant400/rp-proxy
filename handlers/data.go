// handlers/data.go
package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"text/template"
	"time"

	"github.com/kirant400/rp-wrapper/cache"
	"github.com/kirant400/rp-wrapper/config"

	"github.com/gin-gonic/gin"
)

type templateParams struct {
	PrevSec  string
	EndTime  string
}

// fetchFromLegacy calls the configured upstream URL and returns raw JSON bytes
func fetchFromLegacy(urlTmpl string, params templateParams) (json.RawMessage, error) {
	tmpl, err := template.New("upstream").Parse(urlTmpl)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return nil, err
	}
	upstreamURL := buf.String()

	resp, err := http.Get(upstreamURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, http.ErrMissingFile // use a generic error
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(body), nil
}

// =============================================
// /fill/departments
// =============================================
func FillDepartments(c *gin.Context) {
	baseURL := config.Global.Upstream["departments"]
	if baseURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "departments upstream URL not configured"})
		return
	}

	// Parse base URL to preserve host, path, etc.
	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid upstream URL"})
		return
	}

	var allDepts []map[string]interface{}
	currentParams := parsedBase.Query() // start with initial params (if any)
	page := 1

	for {
		// Set or update page parameter
		currentParams.Set("page", strconv.Itoa(page))

		// Build clean URL with current params
		parsedBase.RawQuery = currentParams.Encode()
		currentURL := parsedBase.String()

		rawPage, hasNext, err := fetchLegacyPage(currentURL)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{
				"error":   "legacy fetch failed",
				"page":    page,
				"url":     currentURL,
				"details": err.Error(),
			})
			return
		}

		// Parse page data
		var pageResp struct {
			Data []map[string]interface{} `json:"data"`
			Next interface{}              `json:"next"`
		}
		if json.Unmarshal(rawPage, &pageResp); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "invalid JSON from legacy", "page": page})
			return
		}

		if len(pageResp.Data) == 0 {
			break // empty page = end
		}

		allDepts = append(allDepts, pageResp.Data...)

		// Stop if next is null or missing
		if hasNext == nil {
			break
		}

		page++ // go to next page
	}

	// Wrap in expected format for StoreDepartments
	finalJSON, _ := json.Marshal(map[string][]map[string]interface{}{"data": allDepts})

	if err := cache.StoreDepartments(finalJSON); err != nil {
		log.Printf("Redis store failed for departments: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        "ok",
		"source":        "legacy (paginated)",
		"total_records": len(allDepts),
		"pages_fetched": page,
		"stored_in":     "redis",
		"key_prefix":    "dept",
		"fetched_at":    time.Now().Unix(),
	})
}

// fetchLegacyPage returns raw body + whether "next" exists (nil = last page)
func fetchLegacyPage(urlStr string) (json.RawMessage, interface{}, error) {
	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var meta struct {
		Next interface{} `json:"next"`
	}
	json.Unmarshal(body, &meta)

	return json.RawMessage(body), meta.Next, nil
}
// =============================================
// /fill/areas
// =============================================
func FillAreas(c *gin.Context) {
	baseURL := config.Global.Upstream["areas"]
	if baseURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "areas upstream URL not configured"})
		return
	}

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid areas upstream URL"})
		return
	}

	var allAreas []map[string]interface{}
	currentParams := parsedBase.Query()
	page := 1

	for {
		currentParams.Set("page", strconv.Itoa(page))
		parsedBase.RawQuery = currentParams.Encode()
		currentURL := parsedBase.String()

		rawPage, hasNext, err := fetchLegacyPage(currentURL)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{
				"error": "legacy fetch failed",
				"page":  page,
				"url":   currentURL,
			})
			return
		}

		var pageResp struct {
			Data []map[string]interface{} `json:"data"`
		}
		if json.Unmarshal(rawPage, &pageResp); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "invalid JSON from legacy (areas)"})
			return
		}

		if len(pageResp.Data) == 0 {
			break
		}

		allAreas = append(allAreas, pageResp.Data...)

		if hasNext == nil {
			break
		}

		page++
	}

	// Wrap for StoreAreas
	finalJSON, _ := json.Marshal(map[string][]map[string]interface{}{"data": allAreas})

	if err := cache.StoreAreas(finalJSON); err != nil {
		log.Printf("Redis store failed for areas: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        "ok",
		"source":        "legacy (paginated)",
		"total_records": len(allAreas),
		"pages_fetched": page,
		"stored_in":     "redis",
		"key_prefix":    "area",
		"fetched_at":    time.Now().Unix(),
	})
}

// =============================================
// /fill/positions
// =============================================
func FillPositions(c *gin.Context) {
	baseURL := config.Global.Upstream["positions"]
	if baseURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "positions upstream URL not configured"})
		return
	}

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid positions upstream URL"})
		return
	}

	var allPositions []map[string]interface{}
	currentParams := parsedBase.Query()
	page := 1

	for {
		currentParams.Set("page", strconv.Itoa(page))
		parsedBase.RawQuery = currentParams.Encode()
		currentURL := parsedBase.String()

		rawPage, hasNext, err := fetchLegacyPage(currentURL)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{
				"error": "legacy fetch failed",
				"page":  page,
				"url":   currentURL,
				"details": err.Error(),
			})
			return
		}

		var pageResp struct {
			Data []map[string]interface{} `json:"data"`
		}
		if json.Unmarshal(rawPage, &pageResp); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "invalid JSON from legacy (positions)"})
			return
		}

		if len(pageResp.Data) == 0 {
			break
		}

		allPositions = append(allPositions, pageResp.Data...)

		if hasNext == nil {
			break
		}

		page++
	}

	// Wrap for StorePositions
	finalJSON, _ := json.Marshal(map[string][]map[string]interface{}{"data": allPositions})

	if err := cache.StorePositions(finalJSON); err != nil {
		log.Printf("Redis store failed for positions: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        "ok",
		"source":        "legacy (paginated)",
		"total_records": len(allPositions),
		"pages_fetched": page,
		"stored_in":     "redis",
		"key_prefix":    "pos",
		"fetched_at":    time.Now().Unix(),
	})
}

// =============================================
// /fill/employees
// =============================================
func FillEmployees(c *gin.Context) {
	baseURL := config.Global.Upstream["employees"]
	if baseURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "employees upstream URL not configured"})
		return
	}

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid employees upstream URL"})
		return
	}

	var allEmployees []map[string]interface{}
	currentParams := parsedBase.Query()
	page := 1

	for {
		currentParams.Set("page", strconv.Itoa(page))
		parsedBase.RawQuery = currentParams.Encode()
		currentURL := parsedBase.String()

		rawPage, hasNext, err := fetchLegacyPage(currentURL)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{
				"error": "legacy fetch failed",
				"page":  page,
				"url":   currentURL,
				"details": err.Error(),
			})
			return
		}

		var pageResp struct {
			Data []map[string]interface{} `json:"data"`
		}
		if json.Unmarshal(rawPage, &pageResp); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "invalid JSON from legacy (employees)"})
			return
		}

		if len(pageResp.Data) == 0 {
			break
		}

		allEmployees = append(allEmployees, pageResp.Data...)

		if hasNext == nil {
			break
		}

		page++
	}

	// Wrap for StoreEmployees
	finalJSON, _ := json.Marshal(map[string][]map[string]interface{}{"data": allEmployees})

	if err := cache.StoreEmployees(finalJSON); err != nil {
		log.Printf("Redis store failed for employees: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        "ok",
		"source":        "legacy (paginated)",
		"total_records": len(allEmployees),
		"pages_fetched": page,
		"stored_in":     "redis",
		"key_prefix":    "emp",
		"fetched_at":    time.Now().Unix(),
	})
}

// =============================================
// /fill/trans/:prevsec[/:endtime]
// =============================================
func FillTransactions(c *gin.Context) {
	prevSecStr := c.Param("prevsec")
	endTimeParam := c.Param("endtime")

	if prevSecStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prevsec (seconds back) is required"})
		return
	}

	secondsBack, err := strconv.ParseInt(prevSecStr, 10, 64)
	if err != nil || secondsBack < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prevsec must be a positive integer"})
		return
	}

	// End time: now or manual
	var endTime time.Time
	if endTimeParam != "" {
		endTime, err = time.Parse("2006-01-02 15:04:05", endTimeParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid endtime format"})
			return
		}
	} else {
		endTime = time.Now()
	}

	startTime := endTime.Add(-time.Duration(secondsBack) * time.Second)

	baseURL := config.Global.Upstream["transactions"]
	if baseURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "transactions upstream not configured"})
		return
	}

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid transactions URL"})
		return
	}

	query := parsedBase.Query()
	query.Set("start_time", startTime.Format("2006-01-02 15:04:05"))
	query.Set("end_time", endTime.Format("2006-01-02 15:04:05"))
	parsedBase.RawQuery = query.Encode()

	page := 1
	totalProcessed := 0

	for {
		currentQuery := parsedBase.Query()
		currentQuery.Set("page", strconv.Itoa(page))
		parsedBase.RawQuery = currentQuery.Encode()
		currentURL := parsedBase.String()

		rawPage, hasNext, err := fetchLegacyPage(currentURL)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{
				"error": "legacy fetch failed",
				"page":  page,
				"url":   currentURL,
			})
			return
		}

		// ←←← ONLY CALL store.go function ←←←
		if err := cache.StoreAttendanceFromTransactions(rawPage); err != nil {
			log.Printf("Failed to store attendance page %d: %v", page, err)
		}

		// Count records for response
		var temp struct {
			Data []struct{} `json:"data"`
		}
		json.Unmarshal(rawPage, &temp)
		totalProcessed += len(temp.Data)

		if hasNext == nil {
			break
		}
		page++
	}

	c.JSON(http.StatusOK, gin.H{
		"status":            "ok",
		"source":            "legacy (paginated)",
		"seconds_back":      secondsBack,
		"start_time":        startTime.Format("2006-01-02 15:04:05"),
		"end_time":          endTime.Format("2006-01-02 15:04:05"),
		"pages_fetched":     page,
		"records_processed": totalProcessed,
		"stored_using":      "StoreAttendanceFromTransactions",
		"fetched_at":        time.Now().Unix(),
	})
}