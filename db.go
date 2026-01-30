package main

import (
	"fmt"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type mysqlConfig struct {
	Host   string
	Port   int
	User   string
	Pass   string
	DBName string
}

type CrAgentRun struct {
	ID                 uint64         `gorm:"primaryKey;autoIncrement;type:bigint unsigned;comment:自增主键"`
	Repo               string         `gorm:"size:128;not null;index:idx_repo_change_reported,priority:1;index:idx_repo_reported,priority:1;uniqueIndex:uk_repo_change_run,priority:1;comment:代码仓库标识，如 org/name"`
	CodeChangeID       string         `gorm:"size:128;not null;index:idx_repo_change_reported,priority:2;uniqueIndex:uk_repo_change_run,priority:2;comment:代码变更ID（PR / Change-Id / Commit 等）"`
	AgentRunID         string         `gorm:"type:char(36);not null;uniqueIndex:uk_repo_change_run,priority:3;comment:一次 agent 运行的全局唯一ID(UUID)"`
	AgentVersion       string         `gorm:"size:64;not null;comment:agent 版本"`
	RulesetVersion     string         `gorm:"size:64;not null;comment:规则集版本"`
	ReportedAt         time.Time      `gorm:"type:datetime(3);not null;index:idx_repo_change_reported,priority:3;index:idx_repo_reported,priority:2;index:idx_reported_at;comment:agent 实际完成并上报时间（UTC）"`
	DiffLines          uint32         `gorm:"type:int unsigned;not null;comment:本次变更涉及的 diff 行数"`
	TriggeredTotalHits uint32         `gorm:"type:int unsigned;not null;comment:本次运行命中的规则总数"`
	RuleHitsJSON       datatypes.JSON `gorm:"type:json;not null;comment:各规则命中次数快照，如 {\"RULE-1\":3}"`
	CreatedAt          time.Time      `gorm:"type:datetime(3);autoCreateTime:milli;comment:记录入库时间"`
}

func (CrAgentRun) TableName() string {
	return "cr_agent_run"
}

type CrAgentRunRule struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement;type:bigint unsigned;comment:自增主键"`
	RunID          uint64    `gorm:"type:bigint unsigned;not null;uniqueIndex:uk_run_rule,priority:1;comment:关联 cr_agent_run.id"`
	Repo           string    `gorm:"size:128;not null;index:idx_rule_repo_time,priority:2;index:idx_change_rule,priority:1;comment:仓库标识"`
	CodeChangeID   string    `gorm:"size:128;not null;index:idx_change_rule,priority:2;comment:代码变更ID"`
	ReportedAt     time.Time `gorm:"type:datetime(3);not null;index:idx_rule_time,priority:2;index:idx_rule_repo_time,priority:3;comment:上报时间（UTC）"`
	RulesetVersion string    `gorm:"size:64;not null;index:idx_ruleset_rule,priority:1;comment:规则集版本（冗余自 run）"`
	RuleID         string    `gorm:"size:128;not null;uniqueIndex:uk_run_rule,priority:2;index:idx_rule_time,priority:1;index:idx_rule_repo_time,priority:1;index:idx_change_rule,priority:3;index:idx_ruleset_rule,priority:2;comment:规则ID"`
	HitCount       uint32    `gorm:"type:int unsigned;not null;comment:该规则在本次 run 的命中次数"`
}

func (CrAgentRunRule) TableName() string {
	return "cr_agent_run_rule"
}

type CodeChangeSummary struct {
	Repo               string    `gorm:"size:128;not null;primaryKey;index:idx_repo_last_reported,priority:1;index:idx_repo_run_count,priority:1;comment:仓库标识"`
	CodeChangeID       string    `gorm:"size:128;not null;primaryKey;comment:代码变更ID（PR / Change-Id 等）"`
	RunCount           uint32    `gorm:"type:int unsigned;not null;index:idx_repo_run_count,priority:2;comment:该 code change 被 agent 扫描的次数"`
	FirstReportedAt    time.Time `gorm:"type:datetime(3);not null;comment:首次扫描时间（UTC）"`
	LastReportedAt     time.Time `gorm:"type:datetime(3);not null;index:idx_repo_last_reported,priority:2;comment:最近一次扫描时间（UTC）"`
	MaxTotalHits       uint32    `gorm:"type:int unsigned;not null;comment:历史最大规则命中总数"`
	MaxRunID           uint64    `gorm:"type:bigint unsigned;not null;comment:产生最大命中数的 run_id"`
	MinTotalHits       uint32    `gorm:"type:int unsigned;not null;comment:历史最小规则命中总数"`
	MinRunID           uint64    `gorm:"type:bigint unsigned;not null;comment:产生最小命中数的 run_id"`
	LastRulesetVersion string    `gorm:"size:64;not null;comment:最近一次运行使用的规则集版本"`
	ImprovementRate    *float64  `gorm:"type:decimal(6,5);comment:规则命中改进率，范围 [0,1]，单次 run 时为 NULL"`
}

func (CodeChangeSummary) TableName() string {
	return "code_change_summary"
}

func openDB() (*gorm.DB, error) {
	cfg := mysqlConfig{
		Host:   "bjdd-bcc-ctrl-rdtest-02.bjdd.baidu.com",
		Port:   8979,
		User:   "appuser",
		Pass:   "StrongPass!",
		DBName: "cr-agent",
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User,
		cfg.Pass,
		cfg.Host,
		cfg.Port,
		cfg.DBName,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
