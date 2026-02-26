# Feishu → Codex Runner (Polling)

本项目实现一个本地运行的 Go 服务：轮询飞书消息、解析任务指令、调用本机 Codex CLI 在白名单仓库执行任务，并将摘要回传飞书。

## 功能

- 飞书轮询（无 webhook）
- 指令解析：`#repo=... #branch=... #test_cmd="..."` 或 JSON
- 用户白名单校验（open_id）
- Repo 白名单与分支切换
- Codex CLI 执行 + 测试执行
- 执行结果摘要（输出、diff stat、测试结果）
- 本地 JSON 去重存储（断点续跑）

## 工程结构

- `cmd/runner/main.go`：程序入口
- `internal/feishu`：飞书 token、拉消息、发消息
- `internal/parser`：指令解析
- `internal/repo`：repo 白名单与 git 检查
- `internal/codex`：Codex CLI 调用
- `internal/report`：消息摘要
- `internal/store`：去重状态存储
- `internal/orchestrator`：主流程编排

## 1) 飞书配置

在飞书开放平台创建机器人应用，拿到：

- `FEISHU_APP_ID`
- `FEISHU_APP_SECRET`

并确保应用具备消息读取与发送权限（按飞书 API 权限模型配置）。

## 2) 配置 repos.yaml / allowlist.yaml

### repos.yaml

```yaml
repos:
  - name: aoi-service
    local_path: /Users/me/work/aoi-service
    allowed: true
    default_branch: main
```

### allowlist.yaml

```yaml
open_ids:
  - ou_xxx_user1
  - ou_xxx_user2
```

## 3) 环境变量

```bash
export FEISHU_APP_ID=cli_xxx
export FEISHU_APP_SECRET=xxx
export CODEX_BIN=codex
export RUNNER_POLL_INTERVAL_SEC=8
export RUNNER_WORK_DIR=./runner-data
export RUNNER_REPOS_FILE=./repos.yaml
export RUNNER_ALLOWLIST_FILE=./allowlist.yaml
export RUNNER_DEFAULT_TEST_CMD='go test ./...'
export RUNNER_EXEC_TIMEOUT_MIN=30
```

## 4) 运行

```bash
go run ./cmd/runner
```

## 5) 飞书指令示例

文本格式：

```text
#repo=aoi-service #branch=feat/jwt 添加JWT鉴权中间件，并补单测，跑 go test ./...
#repo=aoi-service #test_cmd="go test ./..." 修复：/healthz 在Redis不可用时返回 503，并写测试
```

JSON 格式：

```json
{"repo":"aoi-service","branch":"feat/jwt","test_cmd":"go test ./...","task":"添加 JWT 鉴权中间件"}
```

## 安全策略（MVP）

- 非 allowlist 用户直接拒绝
- 非 repo 白名单直接拒绝
- repo 有脏工作区时拒绝执行
- 命中危险关键词（如 `rm -rf`）拒绝执行
- 日志截断避免超长回传，完整日志写到本地 `runner-data/logs/`

## 说明与扩展

- 当前先聚焦私聊消息闭环，群聊 @ 可后续扩展。
- 飞书 API 字段可能因权限与版本策略而有差异，如需适配可在 `internal/feishu/client.go` 调整。
