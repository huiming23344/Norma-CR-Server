package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func handleChangeEffectivenessSummary(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		from, to, err := parseTimeRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, errResponse{OK: false, Error: "VALIDATION_ERROR", Message: err.Error()})
			return
		}

		minRuns := parseLimit(c.Query("min_runs"), 2, 1, 1000)
		filtered := applyChangeFilters(db.Model(&CodeChangeSummary{}), c).
			Where("last_reported_at BETWEEN ? AND ?", from, to).
			Where("run_count >= ?", minRuns)

		var totalChanges uint64
		if err := filtered.Select("COUNT(*)").Scan(&totalChanges).Error; err != nil {
			c.JSON(http.StatusInternalServerError, errResponse{OK: false, Error: "INTERNAL_ERROR", Message: err.Error()})
			return
		}

		var improvingChanges uint64
		if err := filtered.Where("max_total_hits > min_total_hits").Select("COUNT(*)").Scan(&improvingChanges).Error; err != nil {
			c.JSON(http.StatusInternalServerError, errResponse{OK: false, Error: "INTERNAL_ERROR", Message: err.Error()})
			return
		}

		var stableChanges uint64
		if err := filtered.Where("max_total_hits = min_total_hits").Select("COUNT(*)").Scan(&stableChanges).Error; err != nil {
			c.JSON(http.StatusInternalServerError, errResponse{OK: false, Error: "INTERNAL_ERROR", Message: err.Error()})
			return
		}

		var avgRate float64
		if err := filtered.Select("COALESCE(AVG(improvement_rate),0)").Scan(&avgRate).Error; err != nil {
			c.JSON(http.StatusInternalServerError, errResponse{OK: false, Error: "INTERNAL_ERROR", Message: err.Error()})
			return
		}

		c.JSON(http.StatusOK, changeEffectivenessSummary{
			OK:                 true,
			From:               from,
			To:                 to,
			TotalChanges:       totalChanges,
			ImprovingChanges:   improvingChanges,
			StableChanges:      stableChanges,
			AvgImprovementRate: avgRate,
		})
	}
}

func handleChangeEffectivenessTop(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		from, to, err := parseTimeRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, errResponse{OK: false, Error: "VALIDATION_ERROR", Message: err.Error()})
			return
		}

		minRuns := parseLimit(c.Query("min_runs"), 2, 1, 1000)
		limit := parseLimit(c.Query("limit"), 5, 1, 50)
		direction := strings.ToLower(strings.TrimSpace(c.Query("direction")))
		if direction != "low" && direction != "high" {
			direction = "high"
		}

		order := "improvement_rate DESC"
		if direction == "low" {
			order = "improvement_rate ASC"
		}

		query := applyChangeFilters(db.Table("code_change_summary"), c).
			Select("repo, code_change_id, run_count, max_total_hits, min_total_hits, (max_total_hits - min_total_hits) AS delta, improvement_rate, last_reported_at, last_ruleset_version").
			Where("last_reported_at BETWEEN ? AND ?", from, to).
			Where("run_count >= ?", minRuns).
			Where("improvement_rate IS NOT NULL")

		var rows []changeEffectivenessRow
		if err := query.Order(order).Limit(limit).Scan(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, errResponse{OK: false, Error: "INTERNAL_ERROR", Message: err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":   true,
			"from": from,
			"to":   to,
			"data": rows,
		})
	}
}

func handleChangeEffectivenessList(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		from, to, err := parseTimeRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, errResponse{OK: false, Error: "VALIDATION_ERROR", Message: err.Error()})
			return
		}

		minRuns := parseLimit(c.Query("min_runs"), 2, 1, 1000)
		limit := parseLimit(c.Query("limit"), 50, 1, 500)
		offset := parseLimit(c.Query("offset"), 0, 0, 100000)

		sort := strings.ToLower(strings.TrimSpace(c.Query("sort")))
		switch sort {
		case "improvement_rate", "delta", "last_reported_at", "run_count", "max_total_hits", "min_total_hits":
		default:
			sort = "improvement_rate"
		}

		order := strings.ToLower(strings.TrimSpace(c.Query("order")))
		if order != "asc" && order != "desc" {
			order = "desc"
		}

		orderExpr := sort + " " + strings.ToUpper(order)
		if sort == "improvement_rate" {
			orderExpr = "improvement_rate IS NULL, improvement_rate " + strings.ToUpper(order)
		}

		query := applyChangeFilters(db.Table("code_change_summary"), c).
			Select("repo, code_change_id, run_count, max_total_hits, min_total_hits, (max_total_hits - min_total_hits) AS delta, improvement_rate, last_reported_at, last_ruleset_version").
			Where("last_reported_at BETWEEN ? AND ?", from, to).
			Where("run_count >= ?", minRuns)

		var rows []changeEffectivenessRow
		if err := query.Order(orderExpr).Order("last_reported_at DESC").Limit(limit).Offset(offset).Scan(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, errResponse{OK: false, Error: "INTERNAL_ERROR", Message: err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":     true,
			"from":   from,
			"to":     to,
			"data":   rows,
			"limit":  limit,
			"offset": offset,
		})
	}
}

func handleChangeEffectivenessRuns(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		codeChangeID := strings.TrimSpace(c.Query("code_change_id"))
		if codeChangeID == "" {
			c.JSON(http.StatusBadRequest, errResponse{OK: false, Error: "VALIDATION_ERROR", Message: "code_change_id is required"})
			return
		}

		limit := parseLimit(c.Query("limit"), 50, 1, 200)
		repo := strings.TrimSpace(c.Query("repo"))
		fromStr := strings.TrimSpace(c.Query("from"))
		toStr := strings.TrimSpace(c.Query("to"))

		var from time.Time
		var to time.Time
		var err error
		if fromStr != "" {
			from, err = parseTimeParam(fromStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, errResponse{OK: false, Error: "VALIDATION_ERROR", Message: err.Error()})
				return
			}
		}
		if toStr != "" {
			to, err = parseTimeParam(toStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, errResponse{OK: false, Error: "VALIDATION_ERROR", Message: err.Error()})
				return
			}
		}
		if !to.IsZero() && !from.IsZero() && to.Before(from) {
			c.JSON(http.StatusBadRequest, errResponse{OK: false, Error: "VALIDATION_ERROR", Message: "to must be >= from"})
			return
		}

		query := db.Model(&CrAgentRun{}).
			Select("reported_at, triggered_total_hits, diff_lines").
			Where("code_change_id = ?", codeChangeID)
		if repo != "" {
			query = query.Where("repo = ?", repo)
		}
		if !from.IsZero() {
			query = query.Where("reported_at >= ?", from)
		}
		if !to.IsZero() {
			query = query.Where("reported_at <= ?", to)
		}

		var rows []changeRunRow
		if err := query.Order("reported_at DESC").Limit(limit).Find(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, errResponse{OK: false, Error: "INTERNAL_ERROR", Message: err.Error()})
			return
		}

		for i := range rows {
			if rows[i].DiffLines > 0 {
				rows[i].HitDensity = float64(rows[i].TriggeredTotalHits) / float64(rows[i].DiffLines)
			}
		}

		for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
			rows[i], rows[j] = rows[j], rows[i]
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":             true,
			"code_change_id": codeChangeID,
			"data":           rows,
		})
	}
}
