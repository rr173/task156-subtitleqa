# 无障碍字幕时轴校对工作台（评测说明）

字幕质检服务：导入转写片段与说话人标记，实时检测时轴重叠、空洞、超速行与描述性字幕缺失；并发编辑产生可解释冲突；发布版本冻结不可编辑，支持版本比较与撤回。

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
go run ./cmd/subtitleqa --smoke-test
go run ./cmd/subtitleqa --addr=:8080 --db=task156-subtitleqa.db
```

## Docker 双架构构建与验证

```bash
bash build_benzhi_docker.sh task156-subtitleqa linux/amd64
docker run --rm task156-subtitleqa /app/subtitleqa --smoke-test

bash build_benzhi_docker.sh task156-subtitleqa linux/arm64
docker run --rm task156-subtitleqa /app/subtitleqa --smoke-test
```

两者均须打印 `smoke test passed: ...` 并以 0 退出。

## --smoke-test 契约

不启动长驻服务；在临时 SQLite 数据库上完整执行：建媒体 → 说话人 → 导入 6 条含问题片段（重叠/空洞/超速/空行/描述缺失/未知说话人）→ 断言 ≥5 条质量项 → 乐观并发编辑成功 → 陈旧版本编辑被拒并记录冲突 → 发布 v1/v2 并比较差异 → 撤回 v1 → **关闭并重开数据库**验证持久化与恢复。全部通过以 0 退出，任一步失败以非 0 退出。

## API 契约

- 路由统一 `/api` 前缀；除根页面外均为 JSON。
- `POST /api/segments/{id}/edit`：`{"actor","base_version",可选字段}`；版本不匹配返回 `409` 且 body 携带 `conflict` 对象（含 `actor_a/actor_b/explanation`）。
- 发布版本为冻结快照：`POST /api/media/{id}/publish` 生成 vN；`POST /api/versions/{id}/withdraw` 仅标记撤回并追加记录，不改写快照。
- 组件版本：Go 1.26.3 / SQLite 3.46.1（modernc.org/sqlite v1.52.0，纯 Go 无 CGO）。
