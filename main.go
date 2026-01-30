package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type agentRunRequest struct {
	Repo               string            `json:"repo"`
	CodeChangeID       string            `json:"code_change_id"`
	AgentRunID         string            `json:"agent_run_id"`
	ReportedAt         time.Time         `json:"reported_at"`
	DiffLines          *uint32           `json:"diff_lines"`
	AgentVersion       string            `json:"agent_version"`
	RulesetVersion     string            `json:"ruleset_version"`
	TriggeredTotalHits uint32            `json:"triggered_total_hits"`
	RuleHits           map[string]uint32 `json:"rule_hits"`
}

type okResponse struct {
	OK           bool   `json:"ok"`
	RunPrimaryID uint64 `json:"run_primary_id"`
	Idempotent   bool   `json:"idempotent"`
}

type errResponse struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

func main() {
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		panic(err)
	}

	db, err := openDB(cfg.MySQL)
	if err != nil {
		panic(err)
	}

	r := gin.Default()
	r.POST("/v1/metrics/agent-runs", func(c *gin.Context) {
		var req agentRunRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, errResponse{OK: false, Error: "VALIDATION_ERROR", Message: err.Error()})
			return
		}

		diffLines, msg := normalizeDiffLines(req.DiffLines)
		if msg != "" {
			c.JSON(http.StatusBadRequest, errResponse{OK: false, Error: "VALIDATION_ERROR", Message: msg})
			return
		}

		runID, idempotent, err := createAgentRun(db, req, diffLines)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errResponse{OK: false, Error: "INTERNAL_ERROR", Message: err.Error()})
			return
		}

		c.JSON(http.StatusOK, okResponse{OK: true, RunPrimaryID: runID, Idempotent: idempotent})
	})

	addr := cfg.Server.Addr
	if addr == "" {
		addr = ":8869"
	}
	if err := r.Run(addr); err != nil {
		panic(err)
	}
}

func normalizeDiffLines(diffLines *uint32) (uint32, string) {
	if diffLines == nil {
		return 0, "diff_lines is required"
	}
	return *diffLines, ""
}

func validateAgentRun(req agentRunRequest) string {
	if strings.TrimSpace(req.Repo) == "" {
		return "repo is required"
	}
	if strings.TrimSpace(req.CodeChangeID) == "" {
		return "code_change_id is required"
	}
	if strings.TrimSpace(req.AgentRunID) == "" {
		return "agent_run_id is required"
	}
	if strings.TrimSpace(req.AgentVersion) == "" {
		return "agent_version is required"
	}
	if strings.TrimSpace(req.RulesetVersion) == "" {
		return "ruleset_version is required"
	}
	if req.ReportedAt.IsZero() {
		return "reported_at is required"
	}
	if req.RuleHits == nil {
		return "rule_hits is required"
	}
	if req.TriggeredTotalHits > 0 && len(req.RuleHits) == 0 {
		return "rule_hits cannot be empty when triggered_total_hits > 0"
	}

	var sum uint64
	for _, v := range req.RuleHits {
		sum += uint64(v)
	}
	if sum != uint64(req.TriggeredTotalHits) {
		return "triggered_total_hits must equal sum(rule_hits)"
	}

	return ""
}

func createAgentRun(db *gorm.DB, req agentRunRequest, diffLines uint32) (uint64, bool, error) {
	var existing CrAgentRun
	query := db.Where("repo = ? AND code_change_id = ? AND agent_run_id = ?", req.Repo, req.CodeChangeID, req.AgentRunID)
	if err := query.First(&existing).Error; err == nil {
		return existing.ID, true, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, false, err
	}

	ruleHitsJSON, err := json.Marshal(req.RuleHits)
	if err != nil {
		return 0, false, err
	}

	var runID uint64
	err = db.Transaction(func(tx *gorm.DB) error {
		run := CrAgentRun{
			Repo:               req.Repo,
			CodeChangeID:       req.CodeChangeID,
			AgentRunID:         req.AgentRunID,
			AgentVersion:       req.AgentVersion,
			RulesetVersion:     req.RulesetVersion,
			ReportedAt:         req.ReportedAt.UTC(),
			DiffLines:          diffLines,
			TriggeredTotalHits: req.TriggeredTotalHits,
			RuleHitsJSON:       datatypes.JSON(ruleHitsJSON),
		}

		if err := tx.Create(&run).Error; err != nil {
			return err
		}

		runID = run.ID
		if len(req.RuleHits) > 0 {
			rules := make([]CrAgentRunRule, 0, len(req.RuleHits))
			for ruleID, count := range req.RuleHits {
				rules = append(rules, CrAgentRunRule{
					RunID:          run.ID,
					Repo:           req.Repo,
					CodeChangeID:   req.CodeChangeID,
					ReportedAt:     req.ReportedAt.UTC(),
					RulesetVersion: req.RulesetVersion,
					RuleID:         ruleID,
					HitCount:       count,
				})
			}
			if err := tx.Create(&rules).Error; err != nil {
				return err
			}
		}

		summary := CodeChangeSummary{
			Repo:               req.Repo,
			CodeChangeID:       req.CodeChangeID,
			RunCount:           1,
			FirstReportedAt:    req.ReportedAt.UTC(),
			LastReportedAt:     req.ReportedAt.UTC(),
			MaxTotalHits:       req.TriggeredTotalHits,
			MaxRunID:           run.ID,
			MinTotalHits:       req.TriggeredTotalHits,
			MinRunID:           run.ID,
			LastRulesetVersion: req.RulesetVersion,
			ImprovementRate:    nil,
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "repo"},
				{Name: "code_change_id"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"run_count":            gorm.Expr("run_count + 1"),
				"first_reported_at":    gorm.Expr("LEAST(first_reported_at, VALUES(first_reported_at))"),
				"last_reported_at":     gorm.Expr("GREATEST(last_reported_at, VALUES(last_reported_at))"),
				"max_total_hits":       gorm.Expr("GREATEST(max_total_hits, VALUES(max_total_hits))"),
				"max_run_id":           gorm.Expr("IF(VALUES(max_total_hits) > max_total_hits, VALUES(max_run_id), max_run_id)"),
				"min_total_hits":       gorm.Expr("LEAST(min_total_hits, VALUES(min_total_hits))"),
				"min_run_id":           gorm.Expr("IF(VALUES(min_total_hits) < min_total_hits, VALUES(min_run_id), min_run_id)"),
				"last_ruleset_version": gorm.Expr("IF(VALUES(last_reported_at) >= last_reported_at, VALUES(last_ruleset_version), last_ruleset_version)"),
				"improvement_rate": gorm.Expr(
					"IF((run_count + 1) >= 2 AND GREATEST(max_total_hits, VALUES(max_total_hits)) > 0," +
						" (GREATEST(max_total_hits, VALUES(max_total_hits)) - LEAST(min_total_hits, VALUES(min_total_hits))) / GREATEST(max_total_hits, VALUES(max_total_hits))," +
						" NULL)",
				),
			}),
		}).Create(&summary).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			if err := query.First(&existing).Error; err == nil {
				return existing.ID, true, nil
			}
		}
		return 0, false, err
	}

	return runID, false, nil
}
