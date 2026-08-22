# task156-subtitleqa · 无障碍字幕时轴校对工作台

字幕质检员在网页加载音视频转写片段和说话人标记，调整时间边界、阅读速度和描述性字幕；系统实时标出重叠、空洞和超速行，并维护**不可编辑的**可发布版本。两个编辑会话修改同一片段时产生**可解释冲突**而非静默覆盖。

## 标准命令

```bash
export GO_BIN=$(command -v go)          # go1.26.3, GOTOOLCHAIN=local
CGO_ENABLED=0 GOTOOLCHAIN=local $GO_BIN build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local $GO_BIN vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local $GO_BIN test  ./...
$GO_BIN run ./cmd/subtitleqa --smoke-test       # 离线端到端自检（含 DB 关闭重开恢复验证）

# 长驻服务
$GO_BIN run ./cmd/subtitleqa --addr=:8080 --db=task156-subtitleqa.db
# 浏览器打开 http://localhost:8080 或调用 /api/... 接口
```

## API 入口（统一 /api 前缀，>20 个）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | /api/stats | 全局统计 |
| POST | /api/media | 创建媒体 |
| GET | /api/media | 媒体列表 |
| GET | /api/media/{id} | 媒体详情 |
| POST | /api/media/{id}/speakers | 添加说话人标记 |
| GET | /api/media/{id}/speakers | 说话人列表 |
| POST | /api/media/{id}/segments | 批量导入字幕片段（序号冲突/非法时轴报错） |
| GET | /api/media/{id}/segments | 片段列表（时轴序） |
| GET | /api/segments/{id} | 片段详情 |
| GET | /api/segments/{id}/revisions | 片段编辑历史 |
| POST | /api/segments/{id}/edit | 提交编辑（base_version 并发控制，冲突返回 409+冲突记录） |
| GET | /api/media/{id}/quality | 质量检查列表 |
| GET | /api/media/{id}/quality/summary | 质量汇总 |
| POST | /api/media/{id}/quality/recompute | 重算质量项 |
| POST | /api/media/{id}/publish | 发布冻结版本 |
| GET | /api/media/{id}/versions | 版本历史 |
| GET | /api/media/{id}/conflicts | 编辑冲突记录 |
| GET | /api/versions/{id} | 版本详情 |
| POST | /api/versions/{id}/withdraw | 撤回版本（快照仍冻结可读） |
| GET | /api/versions/compare?v1=&v2= | 两个版本字段级差异 |
| POST | /api/demo | 载入示例数据并跑通全流程 |
| GET | /api/metrics | 运行计数 |

## 领域规则要点

- **重叠/空洞/超速**：相邻片段 end>start 判重叠；间隔 >1000ms 判空洞；阅读速度 >12 字/秒判超速行。
- **并发编辑**：片段带版本号，编辑必须携带 base_version；不匹配则拒绝写入、记录冲突（含双方操作人与原因），绝不静默覆盖。
- **发布版本不可编辑**：发布即冻结 JSON 快照；后续修订生成新版本；撤回只改标记并追加撤回记录。
- **持久化与恢复**：全部状态存 SQLite（modernc.org/sqlite，纯 Go 无 CGO）；重启后 Recover 幂等重算质量项，发布快照不受影响。
