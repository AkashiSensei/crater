---
name: crater-cli-admin-read
version: 1.1.0
description: "Crater CLI 管理员只读域：指导 AI Agent 通过 crater admin ... 查看平台级只读信息。仅当用户明确要求管理员或平台级资源时使用。"
metadata:
  requires:
    bins: ["crater"]
  cliHelp: "crater admin --help"
---

# Crater CLI 管理员读取

**CRITICAL — 开始前 MUST 先读取 `crater-cli-shared`（可能路径：[`../crater-cli-shared/SKILL.md`](../crater-cli-shared/SKILL.md)），其中包含全局选项、非交互调用、错误处理和敏感信息规则。**

本 Skill 仅用于管理员视图。普通用户可见数据请使用 `crater-cli-read`，不要为了“多拿一点数据”擅自切换到 `crater admin ...`。

## 适用场景

- 用户明确要求查看管理员或平台级账户、资源、数据集、下载、审批单、用户、计费数据。
- 用户明确要求查看系统配置、队列配额、GPU 分析记录、操作日志、定时任务或白名单。
- 用户具备管理员权限，并希望以 `--json --no-interactive` 供脚本或 Agent 消费。

## 安全原则

- 本 Skill 只覆盖只读命令，不执行 update/delete/reconcile 等写操作。
- 管理员命令统一使用 `crater admin ...` 前缀；不要使用普通命令加 `--admin`。
- 如果 API 返回 403/401，向用户说明需要管理员权限或刷新登录态，不要尝试绕过权限。

## 常用范例

```bash
crater admin account ls --json
crater admin account get 1 --json
crater admin account members 1 --json
crater admin resource networks 1 --json
crater admin dataset ls --json
crater admin model-download ls --status Downloading --page-size 15 --json
crater admin billing status --json
crater admin billing jobs --days 7 --search experiment --page-size 15 --json
crater admin job ls --search experiment --page-size 15 --json
crater admin image ls --owner alice --page-size 15 --json
crater admin image build-ls --page-size 15 --json
crater admin order ls --status Pending --creator alice --page-size 15 --json
crater admin user ls --search wang --role User --status Active --page-size 15 --json
crater admin user billing summary --json
crater admin system-config llm --json
crater admin operation-logs --page 1 --limit 20 --json
```

## 排查顺序

1. 先用 `crater auth ls --json` 确认 active credentials，并检查 role 是否具备管理员权限。
2. 需要机器解析时加 `--json --no-interactive`。
3. 采用公共分页的管理员 Job、镜像/构建、计费作业、工单和用户列表默认只返回 `15` 条并包含 `data.pagination`；优先逐页读取，只有明确需要完整结果时才使用 `--all-pages`。`operation-logs` 保留自己的 `--page` / `--limit` 契约，不要套用 `--page-size`。
4. 工单的 `--creator` 仅属于 `crater admin order ls`；普通用户工单列表没有该 flag。分页与筛选值错误会聚合，读取 `context.issues` 一次修正。
5. API 失败时根据 stderr JSON 的 `category`、`code`、`context.http_status` 判断是未登录、无权限、资源不存在还是服务端错误。
