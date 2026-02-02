package main

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func parseTimeRange(c *gin.Context) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	from := now.Add(-7 * 24 * time.Hour)
	to := now

	if v := strings.TrimSpace(c.Query("from")); v != "" {
		parsed, err := parseTimeParam(v)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		from = parsed
	}
	if v := strings.TrimSpace(c.Query("to")); v != "" {
		parsed, err := parseTimeParam(v)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		to = parsed
	}
	if to.Before(from) {
		return time.Time{}, time.Time{}, errInvalidRange
	}
	return from, to, nil
}

var errInvalidRange = errors.New("to must be >= from")

func parseTimeParam(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	if isAllDigits(value) {
		seconds, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(seconds, 0).UTC(), nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func isAllDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}

func parseLimit(raw string, def, min, max int) int {
	if raw == "" {
		return def
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	if parsed < min {
		return min
	}
	if parsed > max {
		return max
	}
	return parsed
}

func applyRunFilters(db *gorm.DB, c *gin.Context) *gorm.DB {
	if v := strings.TrimSpace(c.Query("repo")); v != "" {
		db = db.Where("repo = ?", v)
	}
	if v := strings.TrimSpace(c.Query("ruleset_version")); v != "" {
		db = db.Where("ruleset_version = ?", v)
	}
	if v := strings.TrimSpace(c.Query("agent_version")); v != "" {
		db = db.Where("agent_version = ?", v)
	}
	if v := strings.TrimSpace(c.Query("code_change_id")); v != "" {
		db = db.Where("code_change_id = ?", v)
	}
	return db
}

func applyChangeFilters(db *gorm.DB, c *gin.Context) *gorm.DB {
	if v := strings.TrimSpace(c.Query("repo")); v != "" {
		db = db.Where("repo = ?", v)
	}
	if v := strings.TrimSpace(c.Query("ruleset_version")); v != "" {
		db = db.Where("last_ruleset_version = ?", v)
	}
	if v := strings.TrimSpace(c.Query("code_change_id")); v != "" {
		db = db.Where("code_change_id = ?", v)
	}
	return db
}

func applyRunFilterSQL(sql string, c *gin.Context) string {
	if strings.TrimSpace(c.Query("repo")) != "" {
		sql += " AND repo = ?"
	}
	if strings.TrimSpace(c.Query("ruleset_version")) != "" {
		sql += " AND ruleset_version = ?"
	}
	if strings.TrimSpace(c.Query("agent_version")) != "" {
		sql += " AND agent_version = ?"
	}
	if strings.TrimSpace(c.Query("code_change_id")) != "" {
		sql += " AND code_change_id = ?"
	}
	return sql
}

func buildRunFilterArgs(c *gin.Context) []interface{} {
	args := make([]interface{}, 0, 4)
	if v := strings.TrimSpace(c.Query("repo")); v != "" {
		args = append(args, v)
	}
	if v := strings.TrimSpace(c.Query("ruleset_version")); v != "" {
		args = append(args, v)
	}
	if v := strings.TrimSpace(c.Query("agent_version")); v != "" {
		args = append(args, v)
	}
	if v := strings.TrimSpace(c.Query("code_change_id")); v != "" {
		args = append(args, v)
	}
	return args
}

func loadTotalRuns(db *gorm.DB, from, to time.Time, c *gin.Context) (uint64, error) {
	filtered := applyRunFilters(db.Model(&CrAgentRun{}), c).
		Where("reported_at BETWEEN ? AND ?", from, to)
	var total uint64
	if err := filtered.Select("COUNT(*)").Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func ruleFilterSQL(alias string, c *gin.Context) (string, []interface{}) {
	prefix := ""
	if alias != "" {
		prefix = alias + "."
	}
	parts := make([]string, 0, 3)
	args := make([]interface{}, 0, 3)
	if v := strings.TrimSpace(c.Query("repo")); v != "" {
		parts = append(parts, prefix+"repo = ?")
		args = append(args, v)
	}
	if v := strings.TrimSpace(c.Query("ruleset_version")); v != "" {
		parts = append(parts, prefix+"ruleset_version = ?")
		args = append(args, v)
	}
	if v := strings.TrimSpace(c.Query("rule_id")); v != "" {
		parts = append(parts, prefix+"rule_id = ?")
		args = append(args, v)
	}
	if len(parts) == 0 {
		return "", args
	}
	return " AND " + strings.Join(parts, " AND "), args
}

func buildRuleQualityBaseSQL(c *gin.Context, from, to time.Time) (string, []interface{}) {
	filterA, argsA := ruleFilterSQL("", c)
	filterR, argsR := ruleFilterSQL("r", c)

	aSQL := "SELECT rule_id, COALESCE(SUM(hit_count),0) AS total_hits, COUNT(DISTINCT run_id) AS run_count, MAX(reported_at) AS last_seen_at " +
		"FROM cr_agent_run_rule WHERE reported_at BETWEEN ? AND ?" + filterA + " GROUP BY rule_id"

	rcSQL := "SELECT r.rule_id, r.repo, r.code_change_id, MAX(r.hit_count) AS max_hit, MAX(r.reported_at) AS last_hit_time " +
		"FROM cr_agent_run_rule r WHERE r.reported_at BETWEEN ? AND ?" + filterR + " GROUP BY r.rule_id, r.repo, r.code_change_id"

	tSQL := "SELECT rc.rule_id, rc.repo, rc.code_change_id, rc.max_hit, " +
		"CASE WHEN s.last_reported_at > rc.last_hit_time THEN 0 ELSE r2.hit_count END AS final_hit " +
		"FROM (" + rcSQL + ") rc " +
		"JOIN cr_agent_run_rule r2 ON r2.rule_id = rc.rule_id AND r2.repo = rc.repo AND r2.code_change_id = rc.code_change_id AND r2.reported_at = rc.last_hit_time " +
		"JOIN code_change_summary s ON s.repo = rc.repo AND s.code_change_id = rc.code_change_id " +
		"WHERE s.last_reported_at BETWEEN ? AND ?"

	bSQL := "SELECT rule_id, COUNT(*) AS change_count, " +
		"SUM(CASE WHEN final_hit < max_hit THEN 1 ELSE 0 END) AS fix_count, " +
		"SUM(CASE WHEN final_hit = 0 AND max_hit > 0 THEN 1 ELSE 0 END) AS disappear_count, " +
		"AVG(max_hit - final_hit) AS avg_drop " +
		"FROM (" + tSQL + ") t GROUP BY rule_id"

	mainSQL := "SELECT a.rule_id, a.total_hits, a.run_count, a.last_seen_at, " +
		"COALESCE(b.change_count,0) AS change_count, " +
		"(b.fix_count / NULLIF(b.change_count,0)) AS fix_rate, " +
		"(b.disappear_count / NULLIF(b.change_count,0)) AS disappear_rate, " +
		"b.avg_drop AS avg_drop " +
		"FROM (" + aSQL + ") a LEFT JOIN (" + bSQL + ") b ON a.rule_id = b.rule_id"

	args := []interface{}{from, to}
	args = append(args, argsA...)
	args = append(args, from, to)
	args = append(args, argsR...)
	args = append(args, from, to)

	return mainSQL, args
}

func buildRuleQualityRows(rows []ruleQualityAggRow, totalRuns uint64) []ruleQualityRow {
	resp := make([]ruleQualityRow, 0, len(rows))
	for _, row := range rows {
		hitRate := 0.0
		if totalRuns > 0 {
			hitRate = float64(row.RunCount) / float64(totalRuns)
		}
		var fixRate *float64
		if row.FixRate.Valid {
			value := row.FixRate.Float64
			fixRate = &value
		}
		var disappearRate *float64
		if row.DisappearRate.Valid {
			value := row.DisappearRate.Float64
			disappearRate = &value
		}
		var avgDrop *float64
		if row.AvgDrop.Valid {
			value := row.AvgDrop.Float64
			avgDrop = &value
		}
		resp = append(resp, ruleQualityRow{
			RuleID:        row.RuleID,
			TotalHits:     row.TotalHits,
			RunCount:      row.RunCount,
			HitRate:       hitRate,
			ChangeCount:   row.ChangeCount,
			FixRate:       fixRate,
			DisappearRate: disappearRate,
			AvgDrop:       avgDrop,
			LastSeenAt:    row.LastSeenAt,
		})
	}
	return resp
}

func loadTopVersions(filtered *gorm.DB, column string) ([]versionBucketCount, error) {
	var rows []versionBucketCount
	if err := filtered.Select(column + " AS version, COUNT(*) AS runs").
		Group(column).
		Order("runs DESC").
		Limit(5).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
