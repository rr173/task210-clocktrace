# clocktrace — 网络时间同步根因定位服务

面向网络运维工程师的纯后端服务：接收多节点的时钟同步样本（偏移 / 往返延迟 / 时钟源）与
拓扑快照，沿 NTP/PTP 同步层级计算偏移传播，识别源切换与最早异常链路，输出根因候选与
证据路径；工程师可确认根因、否决候选、标记测量不可信并封存事件。

## 业务闭环

1. **登记拓扑**：创建网络快照，登记同步节点（grandmaster/boundary/ordinary）与层级边。
2. **采集样本**：提交各节点样本（偏移、往返延迟、时钟源），重复样本按节点序号幂等。
3. **根因定位**：锁定快照后分析偏移序列，识别源切换与传播，生成根因候选与证据路径。
4. **裁决**：工程师确认根因、否决候选、标记测量不可信。
5. **封存**：确认后封存事件，封存后只读。

## 快速开始

```bash
# 启动服务
go run ./cmd/clocktrace --addr :8080 --db clocktrace.db

# 端到端自检（关闭重开同一数据库验证持久化恢复）
go run ./cmd/clocktrace --smoke-test --db smoke.db
```

## 构建与验证

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
```

## 主要 API（统一前缀 /api）

| 能力 | 入口 |
|---|---|
| 快照创建/列表/查询/锁定/归档 | `POST /api/snapshots`、`GET /api/snapshots`、`GET /api/snapshots/{id}`、`POST /api/snapshots/{id}/lock`、`POST /api/snapshots/{id}/archive` |
| 节点登记/列表 | `POST /api/snapshots/{id}/nodes`、`GET /api/snapshots/{id}/nodes` |
| 边登记/列表 | `POST /api/snapshots/{id}/links`、`GET /api/snapshots/{id}/links` |
| 拓扑诊断 | `GET /api/snapshots/{id}/inspect` |
| 一键分析 | `POST /api/snapshots/{id}/analyze` |
| 样本提交/查询 | `POST /api/samples`、`GET /api/samples`、`GET /api/samples/{id}` |
| 事件/定位/封存 | `GET /api/events`、`GET /api/events/{id}`、`POST /api/events/{id}/localize`、`POST /api/events/{id}/seal` |
| 候选/证据/裁决 | `GET /api/events/{id}/candidates`、`GET /api/events/{id}/evidence`、`POST /api/candidates/{id}/confirm`、`POST /api/candidates/{id}/reject` |
| 统计/健康 | `GET /api/stats`、`GET /api/health` |

## 持久化

SQLite（modernc.org/sqlite v1.52.0，纯 Go 无 CGO）。建表：`snapshots`、`nodes`、`links`、
`samples`、`drift_events`、`root_cause_candidates`、`evidence_paths`、`verdicts`。
样本按 `UNIQUE(node_key, sequence)` 幂等；封存事件只读。
