# 修复：过长行质检告警缺失

## 问题

极长的一行字幕没有在质检结果中标成过长行（`long_line`），审核人员会漏掉难以阅读的排版。超过可读长度的字幕应稳定显示对应告警。

根因是**三处叠加缺陷**，每一层都把上一层的结果抹掉，导致 `long_line` 告警从生成到展示全链路失效：

1. **`internal/timeline/analysis.go:96`** — 阈值写成 `cfg.MaxSegmentChars*100`（=`84×100=8400` 字符），而非 `cfg.MaxSegmentChars`。一行要超过 **8400** 字才触发，但告警消息又写的是 "max 84"。300 字的超长行根本不报警。
2. **`internal/qa/qa.go:31-34`** — `Recompute` 主动 `delete`/过滤掉 `RuleLongLine` 的 finding，导致引擎算出也落不进 `quality_checks` 表、`/quality` 列表里看不到。
3. **`internal/service/service.go:318`** — `QualitySummary` 又 `delete(summary.ByRule, timeline.RuleLongLine)`，汇总里也消失。

对照同文件里 overlap/gap/overspeed 的处理方式（引擎直接生成 → 全量持久化 → 结构化字段+`ByRule`），`long_line` 在每一层都被单独剔除，行为完全不一致。

## 修复方案

按现有 overlap/gap/overspeed 的范式，让 `long_line` 走完整的"生成→持久化→汇总→展示"链路。

### 1. `internal/timeline/analysis.go`（核心阈值）

`if runes > cfg.MaxSegmentChars*100 {` → `if runes > cfg.MaxSegmentChars {`

这是根因修复：阈值恢复为配置值 84，与消息里的 "max 84" 一致。

### 2. `internal/qa/qa.go`（持久化）

删除 `Recompute` 里过滤 `RuleLongLine` 的循环：

```go
kept := findings[:0]
for _, finding := range findings {
    if finding.Rule != timeline.RuleLongLine {
        kept = append(kept, finding)
    }
}
findings = kept
```

整段删掉，让所有 finding（含 `long_line`）正常落库。删除后 `time`/`config`/`model`/`speaker`/`store`/`timeline` 仍都被使用，无 import 需调整。

### 3. `internal/service/service.go`（汇总）

删除 `QualitySummary` 末尾的 `delete(summary.ByRule, timeline.RuleLongLine)` 一行，让 `long_line` 在 `ByRule` 里正常出现。

### 4. `internal/webui/webui.go`（展示）

汇总文案补一个"过长"计数，与"重叠/空洞/超速"并列，避免审核人员在页面上仍看不到长行统计。

在 `model.QualitySummary` 增加结构化字段 `LongLines int`（与 `Overlaps/Gaps/Overspeed` 范式一致），`qa.Summary` 填充，WebUI 文案加上"过长"+`s.long_lines`。

> 这一步顺带把 `long_line` 也纳入像 overspeed 那样的"命名计数字段"，而不是只靠 `ByRule`，保持与现有 overlap/gap/overspeed 同形，符合项目一贯风格。

## 影响面与回归校验

- **smoke test**：demo #3 片段是 31 字，远低于 84，修复后**不**触发 `long_line`，仍走原超速路径，`quality=6` 不变 → smoke 不破。已基线验证 `quality=6`。
- **现有 timeline/qa/service/httpapi 测试**：均不依赖 `long_line` 被删除，修复后行为更正确，不破。已基线全部 `ok`。
- **新增测试**：在 `internal/timeline/analysis_test.go` 增 `TestAnalyzeLongLine`：构造一条 >84 字的字幕，断言产生 `RuleLongLine` + `SeverityWarning`；以及一条 ≤84 字的行**不**产生该 finding，锁死阈值不再回到 `*100`。在 `internal/qa/qa_test.go` 或 service 层断言 `long_line` 落进 `Quality()` 列表与 `QualitySummary.ByRule["long_line"]`。

## 验证命令

```bash
export GO_BIN=$(command -v go)
CGO_ENABLED=0 GOTOOLCHAIN=local $GO_BIN build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local $GO_BIN vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local $GO_BIN test  ./...
$GO_BIN run ./cmd/subtitleqa --smoke-test
```

全绿、smoke 打印 `... quality=6 ...` 即完成。
