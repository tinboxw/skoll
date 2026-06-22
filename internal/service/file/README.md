# internal/service/file

## 功能说明

文件服务层，协调 file metadata repository 与 object store port。

## Upload 一致性规则

1. 先构造并校验 `FileObject` pending metadata。
2. 先写入 object store。
3. object 写入成功后，将 metadata 标记为 `available` 并写入 repository。
4. metadata 写入失败时删除已写入的 object，避免幽灵对象。
5. object 写入失败时不写 metadata。

## Access 权限策略

`Service.AuthorizeAccess` 对非 public 文件默认拒绝。

1. `public` 文件在状态为 `available` 时可下载。
2. `private` 文件允许 owner 下载；非 owner 必须通过 RBAC resource `file:private` 与 action `download`。
3. `plugin_asset` 文件必须通过 RBAC resource `plugin:<plugin_id>:asset` 与 action `download`；没有 plugin id 时回退到 `file:plugin_asset`。
4. 非 `available` 状态文件在 owner 和 RBAC 检查前直接拒绝。

本包不处理 HTTP multipart、审计事件或本地/S3 具体实现细节。
