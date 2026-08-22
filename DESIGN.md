# task156-subtitleqa 设计文档

> 基于需求池 REQ-20260822-021「无障碍字幕时轴校对工作台」建立。
> 领取时间：2026-08-22；生成目录：`task0822/task156-subtitleqa/env`。

## 1. 业务闭环

字幕质检员为一段音视频（媒体）登记说话人标记并批量导入带序号的转写片段。系统按时间轴自动派生质量项（重叠、空洞、超速行、空行、超长行、描述性字幕缺失、未知说话人）。质检员以 `base_version` 乐观并发方式编辑片段；编辑成功立即重算质量项。质检通过后可发布**冻结快照版本**，后续修订只能生成新版本；已发布版本支持撤回（保留快照）与两版本字段级比较。两个会话并发编辑同一片段时，版本不匹配的编辑被拒绝并记录可解释冲突，绝不静默覆盖。

## 2. 核心实体与状态

| 实体 | 状态集 | 说明 |
| --- | --- | --- |
| Media | draft / ready / published / archived | 媒体元数据（语言、时长、帧率、来源、校验和） |
| Speaker | — | 说话人标记（label/color），挂靠媒体 |
| Segment | draft / proofed / conflict / published | 单条字幕：index、start_ms、end_ms、text、speaker_id、is_descriptive、language、**version**（乐观锁） |
| Revision | — | 编辑审计：actor、op_type、base_version、before/after JSON |
| QualityCheck | resolved bool | 规则、严重级别、消息、检出时间 |
| PublishVersion | frozen / withdrawn | 冻结快照 JSON + 版本号 |
| Withdrawal | — | 撤回记录（actor、reason） |
| Conflict | — | 被拒并发编辑：actor_a/actor_b、base/attempted version、explanation |

## 3. API 形态

统一 `/api` 前缀 JSON API（>20 个）：媒体 CRUD、说话人管理、片段批量导入/查询/编辑/历史、质量列表/汇总/重算、发布/版本列表/详情/撤回/比较、冲突记录、统计、示例载入与自检；根路径提供由后端数据驱动的轻量 Web 页。

## 4. 数据模型与持久化

- 单文件 SQLite（`modernc.org/sqlite` v1.52.0，纯 Go、无 CGO、离线可构建；SQLite 3.46.1）。
- 表：media、speakers、segments、revisions、quality_checks、publish_versions、withdrawals、conflicts；含索引。
- 连接池限 1 个写连接，天然串行化 SQLite 写入。
- **重启恢复**：所有状态落库；启动时 `Recover` 对非归档媒体幂等重算质量项，发布快照不受影响。`--smoke-test` 通过关闭/重开同一 DB 验证持久化与恢复。

## 5. 状态/并发约束

- **乐观并发编辑**：片段带 `version`；`EditSegment` 要求 `base_version == version`，否则拒绝写入 → 记录 Conflict（含双方 actor 与原因）→ 返回 409；允许客户端重拉重提。
- **序号冲突**：导入时 `index` 必须唯一且 ≥0，重复即报错；时间轴非法（end<=start、负值）报错。
- **发布不可变**：发布即冻结 JSON 快照；撤回仅置 withdrawn 标记并追加 Withdrawal 记录；比较只读。
- **质量重算幂等**：先删后插，同一输入产生同一结果。

## 6. 模块责任

- `store`：SQLite 建表迁移 + 各实体 CRUD。
- `timeline`：时轴分析纯函数（重叠/空洞/超速/空行/超长/描述缺失/未知说话人）。
- `speaker`：说话人集合/覆盖度/孤儿片段。
- `revision`：乐观补丁应用、op 分类、变更描述。
- `publish`：快照构建/解析/字段级比较。
- `qa`：编排重算 + 汇总。
- `service`：校验、门禁、编排门面。
- `httpapi` / `webui` / `demo` / `metrics` / `config`：HTTP、轻量页面、自检、计数、阈值。
