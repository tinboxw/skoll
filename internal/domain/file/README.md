# internal/domain/file

## 功能说明

文件与对象存储领域包，承载文件元数据的 canonical 模型。当前包只定义领域对象和值对象，不绑定本地文件系统、S3、HTTP 上传、数据库模型或前端实现。

## 当前规划文件

- `doc.go`
- `object.go`
- `rules.go`
- `store.go`
- `store_rules.go`
- `multipart.go`
- `multipart_rules.go`

## FileObject

`FileObject` 描述一个可持久化的文件元数据对象:

- `ID`: 文件元数据记录 ID。
- `Key`: 对象存储 key，使用小写规范化，不允许绝对路径、反斜杠、空路径段或相对路径段。
- `Name`: 原始文件名或展示名。
- `Size`: 文件大小，允许零字节，不允许负数。
- `MIME`: 标准 MIME 类型。
- `Hash`: 内容哈希，必须存在且不能包含空白字符。
- `Owner`: 所属主体，包含 `Type` 和 `ID`。
- `Visibility`: `private`、`public` 或 `plugin_asset`。
- `StorageDriver`: 存储驱动标识，例如 `local` 或后续稳定 port 后的 `s3`。
- `Status`: `pending`、`available`、`failed` 或 `deleted`。
- `Source`: 来源模块与可选插件 ID。
- `Metadata`: 可选扩展 metadata，key 使用小写规范化。
- `Meta`: 创建与更新时间。

## 无兼容规则

- 不添加旧上传路径、旧对象 key 或旧 provider 字段的兼容层。
- domain 包不依赖 store、service、handler、GORM model、HTTP multipart 或具体存储 SDK。
- 本地存储和 S3 协议存储必须通过后续 object-store port 接入，不能反向污染本包模型。

## ObjectStore Port

`ObjectStore` 是对象内容读写的稳定端口，当前只覆盖单对象能力:

- `Put`: 写入对象内容，输入包含 key、size、mime、hash、body 和 metadata。
- `Get`: 读取对象内容，返回 `ObjectStream`，调用方负责关闭 `Body`。
- `Delete`: 删除对象内容。
- `Stat`: 查询对象元信息。
- `Presign`: 生成 get/put/delete 预签名访问结果。

该 port 不引入本地文件系统、S3 SDK、HTTP handler 或 multipart 语义；分片上传将在独立 port 中定义。

## MultipartStore Port

`MultipartStore` 是分片上传的稳定端口:

- `Init`: 初始化上传，输入包含 key、size、mime、expected hash 和 metadata。
- `UploadPart`: 上传单个分片，输入包含 upload id、key、part number、size、hash 和 body。
- `Complete`: 完成上传，输入包含 expected size、expected hash 和全部 part hash/etag 上下文。
- `Abort`: 中止上传。

该 port 只描述领域边界和校验上下文，不绑定具体分片协议、临时目录布局或 S3 multipart SDK。
