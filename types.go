package main

import (
	"database/sql"
	"time"
)

type summaryResponse struct {
	OK                bool                 `json:"ok"`
	From              time.Time            `json:"from"`
	To                time.Time            `json:"to"`
	TotalRuns         uint64               `json:"total_runs"`
	TotalHits         uint64               `json:"total_hits"`
	TotalDiffLines    uint64               `json:"total_diff_lines"`
	AvgHitDensity     float64              `json:"avg_hit_density"`
	ActiveRepos       uint64               `json:"active_repos"`
	TopRulesetVersion []versionBucketCount `json:"top_ruleset_versions"`
	TopAgentVersion   []versionBucketCount `json:"top_agent_versions"`
}

type versionBucketCount struct {
	Version string `json:"version"`
	Runs    uint64 `json:"runs"`
}

type timeSeriesPoint struct {
	Bucket string  `json:"bucket"`
	Value  float64 `json:"value"`
}

type recentRunRow struct {
	ID                 uint64    `json:"id"`
	Repo               string    `json:"repo"`
	CodeChangeID       string    `json:"code_change_id"`
	AgentRunID         string    `json:"agent_run_id"`
	ReportedAt         time.Time `json:"reported_at"`
	DiffLines          uint32    `json:"diff_lines"`
	TriggeredTotalHits uint32    `json:"triggered_total_hits"`
	AgentVersion       string    `json:"agent_version"`
	RulesetVersion     string    `json:"ruleset_version"`
	HitDensity         float64   `json:"hit_density"`
}

type topRuleRow struct {
	RuleID    string `json:"rule_id"`
	TotalHits uint64 `json:"total_hits"`
	RunCount  uint64 `json:"run_count"`
}

type changeEffectivenessSummary struct {
	OK                 bool      `json:"ok"`
	From               time.Time `json:"from"`
	To                 time.Time `json:"to"`
	TotalChanges       uint64    `json:"total_changes"`
	ImprovingChanges   uint64    `json:"improving_changes"`
	StableChanges      uint64    `json:"stable_changes"`
	AvgImprovementRate float64   `json:"avg_improvement_rate"`
}

type changeEffectivenessRow struct {
	Repo               string    `json:"repo"`
	CodeChangeID       string    `json:"code_change_id"`
	RunCount           uint32    `json:"run_count"`
	MaxTotalHits       uint32    `json:"max_total_hits"`
	MinTotalHits       uint32    `json:"min_total_hits"`
	Delta              uint32    `json:"delta"`
	ImprovementRate    *float64  `json:"improvement_rate"`
	LastReportedAt     time.Time `json:"last_reported_at"`
	LastRulesetVersion string    `json:"last_ruleset_version"`
}

type changeRunRow struct {
	ReportedAt         time.Time `json:"reported_at"`
	TriggeredTotalHits uint32    `json:"triggered_total_hits"`
	DiffLines          uint32    `json:"diff_lines"`
	HitDensity         float64   `json:"hit_density"`
}

type ruleQualitySummary struct {
	OK                bool      `json:"ok"`
	From              time.Time `json:"from"`
	To                time.Time `json:"to"`
	TotalRules        uint64    `json:"total_rules"`
	AvgFixRate        float64   `json:"avg_fix_rate"`
	AvgDisappearRate  float64   `json:"avg_disappear_rate"`
	AvgHitRate        float64   `json:"avg_hit_rate"`
	TotalRunsInRange  uint64    `json:"total_runs"`
	TotalActiveRules  uint64    `json:"total_active_rules"`
	TotalHitCount     uint64    `json:"total_hit_count"`
	TotalRuleRunCount uint64    `json:"total_rule_run_count"`
}

type ruleQualityRow struct {
	RuleID        string    `json:"rule_id"`
	TotalHits     uint64    `json:"total_hits"`
	RunCount      uint64    `json:"run_count"`
	HitRate       float64   `json:"hit_rate"`
	ChangeCount   uint64    `json:"change_count"`
	FixRate       *float64  `json:"fix_rate"`
	DisappearRate *float64  `json:"disappear_rate"`
	AvgDrop       *float64  `json:"avg_drop"`
	LastSeenAt    time.Time `json:"last_seen_at"`
}

type ruleQualityAggRow struct {
	RuleID        string          `json:"rule_id"`
	TotalHits     uint64          `json:"total_hits"`
	RunCount      uint64          `json:"run_count"`
	LastSeenAt    time.Time       `json:"last_seen_at"`
	ChangeCount   uint64          `json:"change_count"`
	FixRate       sql.NullFloat64 `json:"fix_rate"`
	DisappearRate sql.NullFloat64 `json:"disappear_rate"`
	AvgDrop       sql.NullFloat64 `json:"avg_drop"`
}

type ruleQualityTrendPoint struct {
	Bucket string `json:"bucket"`
	Value  uint64 `json:"value"`
}
