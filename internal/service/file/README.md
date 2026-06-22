# internal/service/file

## 功能说明

文件服务层，协调 file metadata repository 与 object store port。

## Upload 一致性规则

1. 先构造并校验 `FileObject` pending metadata。
2. 先写入 object store。
3. object 写入成功后，将 metadata 标记为 `available` 并写入 repository。
4. metadata 写入失败时删除已写入的 object，避免幽灵对象。
5. object 写入失败时不写 metadata。

本包不处理 HTTP multipart、权限策略、审计事件或本地/S3 具体实现细节。
