package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func handleSummary(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		from, to, err := parseTimeRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, errResponse{OK: false, Error: "VALIDATION_ERROR", Message: err.Error()})
			return
		}

		filtered := applyRunFilters(db.Model(&CrAgentRun{}), c)
		filtered = filtered.Where("reported_at BETWEEN ? AND ?", from, to)

		var totals struct {
			TotalRuns      uint64
			TotalHits      uint64
			TotalDiffLines uint64
		}
		if err := filtered.Select("COUNT(*) AS total_runs, COALESCE(SUM(triggered_total_hits),0) AS total_hits, COALESCE(SUM(diff_lines),0) AS total_diff_lines").Scan(&totals).Error; err != nil {
			c.JSON(http.StatusInternalServerError, errResponse{OK: false, Error: "INTERNAL_ERROR", Message: err.Error()})
			return
		}

		var activeRepos uint64
		if err := filtered.Select("COUNT(DISTINCT repo)").Scan(&activeRepos).Error; err != nil {
			c.JSON(http.StatusInternalServerError, errResponse{OK: false, Error: "INTERNAL_ERROR", Message: err.Error()})
			return
		}

		topRuleset, err := loadTopVersions(filtered, "ruleset_version")
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResponse{OK: false, Error: "INTERNAL_ERROR", Message: err.Error()})
			return
		}

		topAgent, err := loadTopVersions(filtered, "agent_version")
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResponse{OK: false, Error: "INTERNAL_ERROR", Message: err.Error()})
			return
		}

		avgDensity := 0.0
		if totals.TotalDiffLines > 0 {
			avgDensity = float64(totals.TotalHits) / float64(totals.TotalDiffLines)
		}

		c.JSON(http.StatusOK, summaryResponse{
			OK:                true,
			From:              from,
			To:                to,
			TotalRuns:         totals.TotalRuns,
			TotalHits:         totals.TotalHits,
			TotalDiffLines:    totals.TotalDiffLines,
			AvgHitDensity:     avgDensity,
			ActiveRepos:       activeRepos,
			TopRulesetVersion: topRuleset,
			TopAgentVersion:   topAgent,
		})
	}
}

func handleTimeseries(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		from, to, err := parseTimeRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, errResponse{OK: false, Error: "VALIDATION_ERROR", Message: err.Error()})
			return
		}

		metric := strings.ToLower(strings.TrimSpace(c.Query("metric")))
		if metric == "" {
			metric = "runs"
		}
		if metric != "runs" && metric != "hits" && metric != "density" {
			c.JSON(http.StatusBadRequest, errResponse{OK: false, Error: "VALIDATION_ERROR", Message: "metric must be runs|hits|density"})
			return
		}

		bucket := strings.ToLower(strings.TrimSpace(c.Query("bucket")))
		if bucket == "" {
			bucket = "hour"
		}
		if bucket != "hour" && bucket != "day" {
			c.JSON(http.StatusBadRequest, errResponse{OK: false, Error: "VALIDATION_ERROR", Message: "bucket must be hour|day"})
			return
		}

		bucketExpr := "DATE(reported_at)"
		if bucket == "hour" {
			bucketExpr = "DATE_FORMAT(reported_at, '%Y-%m-%d %H:00:00')"
		}

		valueExpr := "COUNT(*)"
		switch metric {
		case "hits":
			valueExpr = "COALESCE(SUM(triggered_total_hits),0)"
		case "density":
			valueExpr = "COALESCE(SUM(triggered_total_hits) / NULLIF(SUM(diff_lines),0), 0)"
		}

		var rows []timeSeriesPoint
		raw := "SELECT " + bucketExpr + " AS bucket, " + valueExpr + " AS value FROM cr_agent_run WHERE reported_at BETWEEN ? AND ?"
		raw = applyRunFilterSQL(raw, c)
		raw += " GROUP BY bucket ORDER BY bucket ASC"
		args := append([]interface{}{from, to}, buildRunFilterArgs(c)...)
		if err := db.Raw(raw, args...).Scan(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, errResponse{OK: false, Error: "INTERNAL_ERROR", Message: err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":     true,
			"from":   from,
			"to":     to,
			"metric": metric,
			"bucket": bucket,
			"data":   rows,
		})
	}
}

func handleRecentRuns(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		from, to, err := parseTimeRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, errResponse{OK: false, Error: "VALIDATION_ERROR", Message: err.Error()})
			return
		}

		limit := parseLimit(c.Query("limit"), 50, 1, 200)
		filtered := applyRunFilters(db.Model(&CrAgentRun{}), c).
			Where("reported_at BETWEEN ? AND ?", from, to)

		var rows []recentRunRow
		if err := filtered.Select("id, repo, code_change_id, agent_run_id, reported_at, diff_lines, triggered_total_hits, agent_version, ruleset_version").
			Order("reported_at DESC").Limit(limit).Find(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, errResponse{OK: false, Error: "INTERNAL_ERROR", Message: err.Error()})
			return
		}

		for i := range rows {
			if rows[i].DiffLines > 0 {
				rows[i].HitDensity = float64(rows[i].TriggeredTotalHits) / float64(rows[i].DiffLines)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":   true,
			"from": from,
			"to":   to,
			"data": rows,
		})
	}
}

func handleTopRules(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		from, to, err := parseTimeRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, errResponse{OK: false, Error: "VALIDATION_ERROR", Message: err.Error()})
			return
		}

		limit := parseLimit(c.Query("limit"), 10, 1, 50)
		repo := strings.TrimSpace(c.Query("repo"))

		query := db.Table("cr_agent_run_rule").
			Select("rule_id, COALESCE(SUM(hit_count),0) AS total_hits, COUNT(*) AS run_count").
			Where("reported_at BETWEEN ? AND ?", from, to)
		if repo != "" {
			query = query.Where("repo = ?", repo)
		}

		var rows []topRuleRow
		if err := query.Group("rule_id").Order("total_hits DESC").Limit(limit).Scan(&rows).Error; err != nil {
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
