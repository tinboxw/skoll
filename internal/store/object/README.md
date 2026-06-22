# internal/store/object

## 功能说明

对象存储适配器实现。当前只提供本地文件系统适配器，用于 M3 文件平台的第一阶段落地。

## LocalStore

`LocalStore` 实现 `internal/domain/file.ObjectStore`:

- `Put`: 写入对象内容并写入同路径 metadata sidecar。
- `Get`: 读取对象内容和 metadata。
- `Delete`: 删除对象内容和 metadata。
- `Stat`: 读取对象 metadata。
- `Presign`: 返回本地语义的 `local://` 预签名占位 URL，不绑定 HTTP 路由。

## 边界规则

- object key 必须通过 `domain/file` 的 key 校验。
- 本地路径通过 `filepath.Rel` 确认仍在 store root 内。
- 本包不处理文件元数据数据库、不做权限判断、不写审计事件。
