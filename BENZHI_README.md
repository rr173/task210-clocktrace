基于 Go 实现的网络时间同步根因定位服务，一款纯后端分析服务，接收多节点时钟同步样本，识别异常根因并发布可追溯诊断结论。

# clocktrace 评测说明

网络时间同步根因定位服务（纯后端）。

## 运行契约

- **启动服务**：`/app/clocktrace --addr :8080 --db clocktrace.db`
- **端到端自检**：`/app/clocktrace --smoke-test --db smoke.db`
  - 真实创建快照与同步拓扑、提交样本、触发根因定位、否决/确认候选、封存事件，
    关闭并重开同一数据库验证持久化与重启恢复，最终以退出码 0 结束。
  - 这是 Docker `CMD` 与双架构验证的唯一判据，**只传 flag，不传路径位置参数**。

## Docker 双架构验证

```bash
# amd64
docker buildx build --platform linux/amd64 --load -t clocktrace:amd64 .
docker run --rm clocktrace:amd64 --smoke-test

# arm64
docker buildx build --platform linux/arm64 --load -t clocktrace:arm64 .
docker run --rm clocktrace:arm64 --smoke-test
```

两项 `docker run` 均须退出码 0。

## 主要 API（前缀 /api）

- 快照：`POST /api/snapshots`、`GET /api/snapshots`、`GET /api/snapshots/{id}`、
  `POST /api/snapshots/{id}/lock`、`POST /api/snapshots/{id}/archive`
- 拓扑：`POST/GET /api/snapshots/{id}/nodes`、`POST/GET /api/snapshots/{id}/links`、
  `GET /api/snapshots/{id}/inspect`
- 分析：`POST /api/snapshots/{id}/analyze`
- 样本：`POST /api/samples`、`GET /api/samples`、`GET /api/samples/{id}`
- 事件：`GET /api/events`、`GET /api/events/{id}`、`POST /api/events/{id}/localize`、
  `POST /api/events/{id}/seal`、`GET /api/events/{id}/candidates`、`GET /api/events/{id}/evidence`
- 候选/裁决：`GET /api/candidates/{id}`、`POST /api/candidates/{id}/confirm`、
  `POST /api/candidates/{id}/reject`、`POST /api/candidates/{id}/untrusted`
- 统计/健康：`GET /api/stats`、`GET /api/health`

## 环境

- Go 1.26.3，`CGO_ENABLED=0`，`GOPROXY=https://goproxy.cn,direct`，`GOSUMDB=sum.golang.google.cn`
- SQLite 驱动 modernc.org/sqlite v1.52.0（纯 Go），SQLite 3.46.1
