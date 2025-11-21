// cache/store.go
package cache

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kirant400/rp-wrapper/callback"
)

// NO ctx declaration here → we use the global ctx from redis.go
// This eliminates the "ctx redeclared in this block" error

// Helper to get string safely
func getString(m map[string]string, key string) string {
	if val, ok := m[key]; ok {
		return val
	}
	return ""
}

// =============================================
// DEPARTMENTS
// =============================================
func StoreDepartments(raw json.RawMessage) error {
    type ParentDept struct {
        ID int `json:"id"`
    }

    type Dept struct {
        ID         int         `json:"id"`
        DeptCode   string      `json:"dept_code"`
        DeptName   string      `json:"dept_name"`
        ParentDept *ParentDept `json:"parent_dept"` // can be null or object
    }

    type Response struct {
        Data []Dept `json:"data"`
    }

    var resp Response
    if err := json.Unmarshal(raw, &resp); err != nil {
        return err
    }

    pipe := Client.Pipeline()
    var ids []string

    for _, d := range resp.Data {
        idStr := fmt.Sprintf("%d", d.ID)
        ids = append(ids, idStr)

        parentID := ""
        if d.ParentDept != nil {
            parentID = fmt.Sprintf("%d", d.ParentDept.ID)
        }

        pipe.HSet(ctx, "dept:"+idStr, map[string]string{
            "name":   d.DeptName,
            "code":   d.DeptCode,
            "parent": parentID,
        })
    }

    if len(ids) > 0 {
        pipe.Set(ctx, "dept", strings.Join(ids, ","), 0)
    }

    _, err := pipe.Exec(ctx)
    return err
}

// =============================================
// AREAS
// =============================================
func StoreAreas(raw json.RawMessage) error {
	type ParentArea struct {
		ID int `json:"id"`
	}

	type Area struct {
		ID            int    `json:"id"`
		AreaCode      string `json:"area_code"`
		AreaName      string `json:"area_name"`
		ParentArea    *struct{} `json:"parent_area"`     // can be object or null
		ParentAreaName *string `json:"parent_area_name"` // can be string or null
	}

	type Response struct {
		Data []Area `json:"data"`
	}

	var resp Response
	if err := json.Unmarshal(raw, &resp); err != nil {
		return err
	}

	pipe := Client.Pipeline()
	var ids []string

	for _, a := range resp.Data {
		idStr := fmt.Sprintf("%d", a.ID)
		ids = append(ids, idStr)

		// Parent is usually null → we store empty string
		parentID := ""

		pipe.HSet(ctx, "area:"+idStr, map[string]string{
			"name":   a.AreaName,
			"code":   a.AreaCode,
			"parent": parentID, // always empty for now, or extend later if needed
		})
	}

	if len(ids) > 0 {
		pipe.Set(ctx, "area", strings.Join(ids, ","), 0)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// =============================================
// POSITIONS
// =============================================
func StorePositions(raw json.RawMessage) error {
	type Position struct {
		ID               int    `json:"id"`
		PositionCode     string `json:"position_code"`
		PositionName     string `json:"position_name"`
		ParentPosition   *struct{} `json:"parent_position"`      // null or object
		ParentPositionName *string `json:"parent_position_name"` // null or string
	}

	type Response struct {
		Data []Position `json:"data"`
	}

	var resp Response
	if err := json.Unmarshal(raw, &resp); err != nil {
		return err
	}

	pipe := Client.Pipeline()
	var ids []string

	for _, p := range resp.Data {
		idStr := fmt.Sprintf("%d", p.ID)
		ids = append(ids, idStr)

		parentID := "" // currently no hierarchy, but ready for future

		pipe.HSet(ctx, "pos:"+idStr, map[string]string{
			"name":   p.PositionName,
			"code":   p.PositionCode,
			"parent": parentID,
		})
	}

	if len(ids) > 0 {
		pipe.Set(ctx, "pos", strings.Join(ids, ","), 0)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// =============================================
// EMPLOYEES
// =============================================
func StoreEmployees(raw json.RawMessage) error {
	type Dept struct {
		ID int `json:"id"`
	}
	type Pos struct {
		ID int `json:"id"`
	}
	type Area struct {
		ID int `json:"id"`
	}
	type Emp struct {
		ID         int     `json:"id"`
		EmpCode    string  `json:"emp_code"`
		FormatName string  `json:"format_name"`
		FullName   string  `json:"full_name"`
		Department *Dept   `json:"department"`
		Position   *Pos    `json:"position"`
		Areas      []Area  `json:"area"`
		HireDate   string  `json:"hire_date"`
		PhotoURL   string  `json:"photo"`
	}

	type Resp struct {
		Data []Emp `json:"data"`
	}

	var resp Resp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return err
	}

	pipe := Client.Pipeline()
	var empIDs []string

	// Temporary maps for building lists
	deptMap := make(map[string][]string)  // dept_id → []emp_id
	areaMap := make(map[string][]string)  // area_id → []emp_id

	for _, e := range resp.Data {
		idStr := strconv.Itoa(e.ID)
		empIDs = append(empIDs, idStr)

		deptID := ""
		if e.Department != nil {
			deptID = strconv.Itoa(e.Department.ID)
		}
		posID := ""
		if e.Position != nil {
			posID = strconv.Itoa(e.Position.ID)
		}

		// Main employee hash
		pipe.HSet(ctx, "emp:"+idStr, map[string]string{
			"code":        e.EmpCode,
			"name":        e.FullName,
			"format_name": e.FormatName,
			"department":  deptID,
			"position":    posID,
			"hire_date":   e.HireDate,
			"photo":   e.PhotoURL,
		})

		// 1. emp_code → emp_id
		if e.EmpCode != "" {
			pipe.Set(ctx, "emp_code:"+e.EmpCode, idStr, 0)
		}

		// 2. Build department → employees list
		if deptID != "" {
			deptMap[deptID] = append(deptMap[deptID], idStr)
		}

		// 3. Build area → employees list
		for _, a := range e.Areas {
			areaID := strconv.Itoa(a.ID)
			areaMap[areaID] = append(areaMap[areaID], idStr)
		}
	}

	// Store main emp list
	if len(empIDs) > 0 {
		pipe.Set(ctx, "emp", strings.Join(empIDs, ","), 0)
	}

	// Store department → employees
	for deptID, emps := range deptMap {
		pipe.Set(ctx, "dept_emps:"+deptID, strings.Join(emps, ","), 0)
	}

	// Store area → employees
	for areaID, emps := range areaMap {
		pipe.Set(ctx, "area_emps:"+areaID, strings.Join(emps, ","), 0)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// =============================================
// TRANSACTIONS
// =============================================
// StoreAttendanceFromTransactions – ONLY attendance state
func StoreAttendanceFromTransactions(raw json.RawMessage) error {
	type Trans struct {
		Emp        int    `json:"emp"`
		PunchTime  string `json:"punch_time"`
		PunchState string `json:"punch_state"`
	}

	type Resp struct {
		Data []Trans `json:"data"`
	}

	var resp Resp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return err
	}

	pipe := Client.Pipeline()

	for _, t := range resp.Data {
		empID := strconv.Itoa(t.Emp)
		keyPrefix := "attn:" + empID

		changed := false
		oldStart := ""
		oldEnd := ""
		if t.PunchState == "0" {
			// Check-In
			oldStart, _ := Client.Get(ctx, keyPrefix+".start").Result()
			if oldStart != t.PunchTime {
				pipe.Set(ctx, keyPrefix+".start", t.PunchTime, 0)
				pipe.Set(ctx, keyPrefix+".status", "present", 0)
				changed = true
			}
		} else {
			// Check-Out
			oldEnd, _ := Client.Get(ctx, keyPrefix+".end").Result()
			if oldEnd != t.PunchTime {
				pipe.Set(ctx, keyPrefix+".end", t.PunchTime, 0)
				pipe.Set(ctx, keyPrefix+".status", "absent", 0)
				changed = true
			}
		}

		// Trigger callback ONLY on real change or first record
		
		if changed || oldStart == "" && oldEnd == "" {
			go func(emp int, punchTime, state string) {
				details := map[string]interface{}{
					"employee_id":   emp,
					"punch_time":    punchTime,
					"punch_state":   state,
					"event_type":    "attendance_update",
					"timestamp":     time.Now().Unix(),
				}
				if state == "0" {
					details["action"] = "check_in"
				} else {
					details["action"] = "check_out"
				}
				callback.SendAttendanceCallback(details)
			}(t.Emp, t.PunchTime, t.PunchState)
		}
	}

	_, err := pipe.Exec(ctx)
	return err
}