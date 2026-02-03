# API 说明

## 通用说明

时间参数：
- `from` 与 `to` 支持 RFC3339 时间字符串（建议 UTC）或 Unix 秒
- 省略时默认最近 7 天（UTC）

通用过滤条件（按接口支持情况提供）：`repo`、`ruleset_version`、`agent_version`、`code_change_id`。

## 指标上报

`POST /v1/metrics/agent-runs`

必填字段：
- `repo` (string)
- `code_change_id` (string)
- `agent_run_id` (string, UUID)
- `reported_at` (RFC3339)
- `diff_lines` (uint32)
- `agent_version` (string)
- `ruleset_version` (string)
- `triggered_total_hits` (uint32)
- `rule_hits` (object: `rule_id -> count`)

示例：

```bash
curl -X POST http://localhost:8869/v1/metrics/agent-runs \
  -H 'Content-Type: application/json' \
  -d '{
    "repo": "org/repo",
    "code_change_id": "PR-123",
    "agent_run_id": "3f8f2f8a-2f0f-4aa6-90a7-7e6b2c1d0d4a",
    "reported_at": "2026-02-03T10:00:00Z",
    "diff_lines": 320,
    "agent_version": "v1.2.3",
    "ruleset_version": "2026.02.01",
    "triggered_total_hits": 5,
    "rule_hits": {
      "RULE-001": 2,
      "RULE-007": 3
    }
  }'
```

响应示例：

```json
{"ok":true,"run_primary_id":123,"idempotent":false}
```

## 汇总与仪表盘接口

`GET /api/summary`
- 参数：`from`、`to`、`repo`、`ruleset_version`、`agent_version`、`code_change_id`

`GET /api/timeseries`
- 参数：`from`、`to`、`metric` (`runs|hits|density`)、`bucket` (`hour|day`)、通用过滤

`GET /api/runs/recent`
- 参数：`from`、`to`、`limit` (1-200)、通用过滤

`GET /api/rules/top`
- 参数：`from`、`to`、`limit` (1-50)、`repo`

## 变更效果分析

`GET /api/change-effectiveness/summary`
- 参数：`from`、`to`、`min_runs` (默认 2)、`repo`、`ruleset_version`、`code_change_id`

`GET /api/change-effectiveness/top`
- 参数：`from`、`to`、`min_runs`、`limit` (1-50)、`direction` (`high|low`)、`repo`、`ruleset_version`、`code_change_id`

`GET /api/change-effectiveness/list`
- 参数：`from`、`to`、`min_runs`、`limit` (1-500)、`offset`、`sort` (`improvement_rate|delta|last_reported_at|run_count|max_total_hits|min_total_hits`)、`order` (`asc|desc`)、`repo`、`ruleset_version`、`code_change_id`

`GET /api/change-effectiveness/runs`
- 参数：`code_change_id` (必填)、`repo`、`from`、`to`、`limit` (1-200)

## 规则质量分析

`GET /api/rule-quality/summary`
- 参数：`from`、`to`、`min_runs` (默认 1)、`min_changes` (默认 2)、`repo`、`ruleset_version`、`rule_id`

`GET /api/rule-quality/top`
- 参数：`from`、`to`、`min_runs`、`min_changes`、`limit` (1-50)、`direction` (`high|low`)、`repo`、`ruleset_version`、`rule_id`

`GET /api/rule-quality/list`
- 参数：`from`、`to`、`min_runs`、`min_changes`、`limit` (1-500)、`offset`、`sort` (`fix_rate|disappear_rate|total_hits|run_count|last_seen_at|avg_drop|change_count`)、`order` (`asc|desc`)、`repo`、`ruleset_version`、`rule_id`

`GET /api/rule-quality/trend`
- 参数：`from`、`to`、`rule_id` (必填)、`bucket` (`hour|day`)、`repo`、`ruleset_version`
