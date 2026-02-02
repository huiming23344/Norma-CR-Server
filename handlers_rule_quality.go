package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func handleRuleQualitySummary(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		from, to, err := parseTimeRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, errResponse{OK: false, Error: "VALIDATION_ERROR", Message: err.Error()})
			return
		}

		minRuns := parseLimit(c.Query("min_runs"), 1, 1, 1000)
		minChanges := parseLimit(c.Query("min_changes"), 2, 1, 1000)

		totalRuns, err := loadTotalRuns(db, from, to, c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResponse{OK: false, Error: "INTERNAL_ERROR", Message: err.Error()})
			return
		}

		baseSQL, args := buildRuleQualityBaseSQL(c, from, to)
		summarySQL := "SELECT COUNT(*) AS total_rules, " +
			"COALESCE(AVG(fix_rate),0) AS avg_fix_rate, " +
			"COALESCE(AVG(disappear_rate),0) AS avg_disappear_rate, " +
			"COALESCE(AVG(run_count),0) AS avg_run_count, " +
			"COALESCE(SUM(total_hits),0) AS total_hit_count, " +
			"COALESCE(SUM(run_count),0) AS total_rule_run_count " +
			"FROM (" + baseSQL + ") q WHERE run_count >= ? AND change_count >= ?"

		args = append(args, minRuns, minChanges)

		var summary struct {
			TotalRules        uint64
			AvgFixRate        float64
			AvgDisappearRate  float64
			AvgRunCount       float64
			TotalHitCount     uint64
			TotalRuleRunCount uint64
		}
		if err := db.Raw(summarySQL, args...).Scan(&summary).Error; err != nil {
			c.JSON(http.StatusInternalServerError, errResponse{OK: false, Error: "INTERNAL_ERROR", Message: err.Error()})
			return
		}

		avgHitRate := 0.0
		if totalRuns > 0 {
			avgHitRate = summary.AvgRunCount / float64(totalRuns)
		}

		c.JSON(http.StatusOK, ruleQualitySummary{
			OK:                true,
			From:              from,
			To:                to,
			TotalRules:        summary.TotalRules,
			AvgFixRate:        summary.AvgFixRate,
			AvgDisappearRate:  summary.AvgDisappearRate,
			AvgHitRate:        avgHitRate,
			TotalRunsInRange:  totalRuns,
			TotalActiveRules:  summary.TotalRules,
			TotalHitCount:     summary.TotalHitCount,
			TotalRuleRunCount: summary.TotalRuleRunCount,
		})
	}
}

func handleRuleQualityTop(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		from, to, err := parseTimeRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, errResponse{OK: false, Error: "VALIDATION_ERROR", Message: err.Error()})
			return
		}

		minRuns := parseLimit(c.Query("min_runs"), 1, 1, 1000)
		minChanges := parseLimit(c.Query("min_changes"), 2, 1, 1000)
		limit := parseLimit(c.Query("limit"), 5, 1, 50)
		direction := strings.ToLower(strings.TrimSpace(c.Query("direction")))
		if direction != "low" && direction != "high" {
			direction = "high"
		}

		order := "fix_rate DESC"
		if direction == "low" {
			order = "fix_rate ASC"
		}

		baseSQL, args := buildRuleQualityBaseSQL(c, from, to)
		topSQL := "SELECT rule_id, total_hits, run_count, last_seen_at, change_count, fix_rate, disappear_rate, avg_drop FROM (" + baseSQL + ") q " +
			"WHERE run_count >= ? AND change_count >= ? ORDER BY " + order + " LIMIT ?"

		args = append(args, minRuns, minChanges, limit)

		var rows []ruleQualityAggRow
		if err := db.Raw(topSQL, args...).Scan(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, errResponse{OK: false, Error: "INTERNAL_ERROR", Message: err.Error()})
			return
		}

		totalRuns, err := loadTotalRuns(db, from, to, c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResponse{OK: false, Error: "INTERNAL_ERROR", Message: err.Error()})
			return
		}

		resp := buildRuleQualityRows(rows, totalRuns)
		c.JSON(http.StatusOK, gin.H{
			"ok":   true,
			"from": from,
			"to":   to,
			"data": resp,
		})
	}
}

func handleRuleQualityList(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		from, to, err := parseTimeRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, errResponse{OK: false, Error: "VALIDATION_ERROR", Message: err.Error()})
			return
		}

		minRuns := parseLimit(c.Query("min_runs"), 1, 1, 1000)
		minChanges := parseLimit(c.Query("min_changes"), 2, 1, 1000)
		limit := parseLimit(c.Query("limit"), 50, 1, 500)
		offset := parseLimit(c.Query("offset"), 0, 0, 100000)

		sort := strings.ToLower(strings.TrimSpace(c.Query("sort")))
		switch sort {
		case "fix_rate", "disappear_rate", "total_hits", "run_count", "last_seen_at", "avg_drop", "change_count":
		default:
			sort = "fix_rate"
		}

		order := strings.ToLower(strings.TrimSpace(c.Query("order")))
		if order != "asc" && order != "desc" {
			order = "desc"
		}

		orderExpr := sort + " " + strings.ToUpper(order)
		if sort == "fix_rate" || sort == "disappear_rate" {
			orderExpr = sort + " IS NULL, " + sort + " " + strings.ToUpper(order)
		}

		baseSQL, args := buildRuleQualityBaseSQL(c, from, to)
		listSQL := "SELECT rule_id, total_hits, run_count, last_seen_at, change_count, fix_rate, disappear_rate, avg_drop FROM (" + baseSQL + ") q " +
			"WHERE run_count >= ? AND change_count >= ? ORDER BY " + orderExpr + " LIMIT ? OFFSET ?"

		args = append(args, minRuns, minChanges, limit, offset)

		var rows []ruleQualityAggRow
		if err := db.Raw(listSQL, args...).Scan(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, errResponse{OK: false, Error: "INTERNAL_ERROR", Message: err.Error()})
			return
		}

		totalRuns, err := loadTotalRuns(db, from, to, c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResponse{OK: false, Error: "INTERNAL_ERROR", Message: err.Error()})
			return
		}

		resp := buildRuleQualityRows(rows, totalRuns)
		c.JSON(http.StatusOK, gin.H{
			"ok":     true,
			"from":   from,
			"to":     to,
			"data":   resp,
			"limit":  limit,
			"offset": offset,
		})
	}
}

func handleRuleQualityTrend(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		from, to, err := parseTimeRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, errResponse{OK: false, Error: "VALIDATION_ERROR", Message: err.Error()})
			return
		}

		ruleID := strings.TrimSpace(c.Query("rule_id"))
		if ruleID == "" {
			c.JSON(http.StatusBadRequest, errResponse{OK: false, Error: "VALIDATION_ERROR", Message: "rule_id is required"})
			return
		}

		bucket := strings.ToLower(strings.TrimSpace(c.Query("bucket")))
		if bucket == "" {
			bucket = "day"
		}
		if bucket != "hour" && bucket != "day" {
			c.JSON(http.StatusBadRequest, errResponse{OK: false, Error: "VALIDATION_ERROR", Message: "bucket must be hour|day"})
			return
		}

		bucketExpr := "DATE(reported_at)"
		if bucket == "hour" {
			bucketExpr = "DATE_FORMAT(reported_at, '%Y-%m-%d %H:00:00')"
		}

		filterSQL, args := ruleFilterSQL("r", c)
		trendSQL := "SELECT " + bucketExpr + " AS bucket, COALESCE(SUM(hit_count),0) AS value FROM cr_agent_run_rule r " +
			"WHERE r.reported_at BETWEEN ? AND ? AND r.rule_id = ?"
		if filterSQL != "" {
			trendSQL += filterSQL
		}
		trendSQL += " GROUP BY bucket ORDER BY bucket ASC"

		allArgs := []interface{}{from, to, ruleID}
		allArgs = append(allArgs, args...)

		var rows []ruleQualityTrendPoint
		if err := db.Raw(trendSQL, allArgs...).Scan(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, errResponse{OK: false, Error: "INTERNAL_ERROR", Message: err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":     true,
			"from":   from,
			"to":     to,
			"bucket": bucket,
			"data":   rows,
		})
	}
}
