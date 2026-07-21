# H5-02 服务端分页与前端大列表验收

> 默认中文；Work Item `H5-02`；验收日期 2026-07-21。

## 验收范围

| 项目 | 结果 |
| --- | --- |
| 数据集 | 员工 125 条、客户 125 条；仅使用 `h5-list-*` ID，运行后残留 `0/0` |
| API | 员工、客户、产品、供应商、仓库统一返回 `items/offset/limit/total/hasMore/nextCursor/sort` |
| 分页稳定性 | 每页 50 条，第二页首项固定为 `H5E0051` / `H5C0051` |
| 虚拟列表 | 每页 50 条仅挂载 24 个可见行，表格高度固定为 520px |
| 请求控制 | 快速筛选取消旧请求；返回第一页命中 10 秒页面缓存 |
| 多语言 | `zh-CN` 默认显示“共 125 条”，`en-US` 显示 “Total 125” |
| 响应式 | 1440x1000 与 390x844 均无页面或分页控件横向溢出 |

`browser-matrix.json` 保存 2 locales x 2 viewports 的机器可读指标，八张 PNG 分别保存员工与客户页面。中文桌面场景额外验证 `H5E0 -> H5E01` 的请求取消，最终结果为 26 条且不出现错误态。

## 复现

先启动连接本地 MySQL 的后端和指向该后端的 Vite 服务，再执行：

```powershell
$env:Path = "D:\workspace\phpEnv\server\mysql\mysql-8.0\bin;" + $env:Path
./scripts/h5-large-list-fixtures.ps1 -Action seed -Rows 125
try {
    python -u ./scripts/h5-large-list-browser.py --base-url http://127.0.0.1:5176
} finally {
    ./scripts/h5-large-list-fixtures.ps1 -Action cleanup
}
```

静态与契约门禁：

```powershell
go test ./internal/repository/pharmaoa ./internal/store/sql/gormrepo ./internal/service/pharmaoa ./internal/handler/http/v1/pharmaoa
npm --prefix web run typecheck
npm --prefix web run build
python -m py_compile ./scripts/h5-large-list-browser.py
```
