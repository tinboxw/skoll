# 插件系统数据库迁移指南

> 本文档说明如何将现有数据库从旧的插件表结构升级到支持系统级/应用级插件的新结构。

## 版本概览

| 版本 | 时间 | 变化 | 兼容性 |
| --- | --- | --- | --- |
| v0 | 早期 | 初始设计（无 level/appId） | 旧插件清单与 code 不兼容 |
| v1 | 当前 | 新增 level/appId/mount_policy 字段 | 自动回填默认值向后兼容 |

## 迁移目标

- ✅ 在 `sk_plugins` 表新增三列：`plugin_level`、`app_id`、`mount_policy`
- ✅ 现有插件记录回填默认值：level=system、mount_policy=admin（符合旧行为）
- ✅ 新安装插件按清单 level/app_id/mount_policy 字段存储
- ✅ 期间保证读/写语义不变，前后端兼容过渡

## SQL 迁移脚本

### 步骤 1：添加新列（可空）

```sql
ALTER TABLE sk_plugins ADD COLUMN plugin_level VARCHAR(32) DEFAULT NULL;
ALTER TABLE sk_plugins ADD COLUMN app_id VARCHAR(128) DEFAULT NULL;
ALTER TABLE sk_plugins ADD COLUMN mount_policy VARCHAR(32) DEFAULT NULL;
```

### 步骤 2：回填现有记录默认值

```sql
-- 系统内置与已安装插件默认为 system/admin
UPDATE sk_plugins 
SET plugin_level = 'system', mount_policy = 'admin'
WHERE plugin_level IS NULL;

-- 若有特定应用级插件标记（比如 source 包含特定前缀），可单独更新
-- 示例：将 source 为 'apps/crm/...' 的插件标记为应用级
UPDATE sk_plugins 
SET plugin_level = 'app', app_id = 'crm'
WHERE source LIKE 'apps/crm/%' AND plugin_level IS NULL;
```

### 步骤 3：设置列约束（可选，增强数据完整性）

```sql
-- 可选：设置非空约束（若确保所有历史记录已回填）
ALTER TABLE sk_plugins MODIFY COLUMN plugin_level VARCHAR(32) NOT NULL DEFAULT 'system';
ALTER TABLE sk_plugins MODIFY COLUMN mount_policy VARCHAR(32) NOT NULL DEFAULT 'admin';

-- app_id 可保持可空（仅应用级插件使用）
-- 或添加检查约束：若 level='app' 则 app_id 不能为空
-- ALTER TABLE sk_plugins ADD CONSTRAINT chk_app_id_required 
--   CHECK ((plugin_level != 'app') OR (app_id IS NOT NULL));
```

## 后端代码适配

### 新增 ORM 映射

文件：`internal/store/sql/gormrepo/plugin_model.go`

```go
type PluginModel struct {
    // ... 已有字段 ...
    PluginLevel  string `gorm:"size:32"`      // 新增
    AppID        string `gorm:"size:128"`     // 新增
    MountPolicy  string `gorm:"size:32"`      // 新增
    // ... 其他字段 ...
}

func PluginModelFromInfo(info plugin.Info) PluginModel {
    return PluginModel{
        // ... 已有字段 ...
        PluginLevel: string(info.Level),
        AppID:       strings.TrimSpace(info.AppID),
        MountPolicy: string(info.MountPolicy),
    }
}

func (m PluginModel) ToInfo() plugin.Info {
    return plugin.Info{
        // ... 已有字段 ...
        Level:       plugin.Level(strings.TrimSpace(m.PluginLevel)),
        AppID:       strings.TrimSpace(m.AppID),
        MountPolicy: plugin.MountPolicy(strings.TrimSpace(m.MountPolicy)),
    }
}
```

**状态**：✅ 已实现

### 新增清单解析

文件：`internal/plugin/types.go`、`internal/plugin/loader.go`

- ✅ 清单解析支持 `level`、`app_id`、`mount_policy` 字段
- ✅ 默认值回填：level→system、mount_policy→admin
- ✅ 自动推导 frontend_entry（按 level 生成）
- ✅ 校验规则：app_id 格式、level/mount_policy 有效值

**状态**：✅ 已实现

### 新增 API 响应字段

文件：`internal/handler/http/v1/plugin/handler.go`

```go
type pluginRecord struct {
    // ... 已有字段 ...
    Level       string `json:"level"`
    AppID       string `json:"appId,omitempty"`
    MountPolicy string `json:"mountPolicy"`
}
```

**状态**：✅ 已实现

## 前端适配

### 类型定义更新

文件：`web/src/plugins/types.ts`

```typescript
export type BackendPluginRecord = {
    // ... 已有字段 ...
    level?: "system" | "app";
    appId?: string;
    mountPolicy?: "admin" | "user" | "mixed";
};

export type FrontendPluginManifest = {
    // ... 已有字段 ...
    level?: "system" | "app";
    appId?: string;
    mountPolicy?: "admin" | "user" | "mixed";
};
```

**状态**：✅ 已实现

### UI 改造

文件：`web/src/views/Plugin/index.vue`

- ✅ 按 level 分组展示（系统级 vs 应用级）
- ✅ 应用级插件下二级分组（按 app_id）
- ✅ 为应用级插件显示 app_id 信息

**状态**：✅ 已实现

### 导航加载

文件：`web/src/plugins/index.ts`

- ✅ 正确推导路由路径：系统级 `/plugins/{id}` vs 应用级 `/apps/{appId}/plugins/{id}`
- ✅ 优先使用后端返回的 frontendEntry，若无则按 level 推导

**状态**：✅ 已实现

## 执行迁移检查清单

- [ ] 备份数据库（生产环境必须）
- [ ] 在测试环境先执行迁移脚本
- [ ] 执行 SQL 步骤 1（添加新列）
- [ ] 执行 SQL 步骤 2（回填默认值）
- [ ] 执行 SQL 步骤 3（可选，添加约束）
- [ ] 重启后端服务（加载新 ORM 映射）
- [ ] 执行 `go test ./...` 验证
- [ ] 在 Web UI 验证插件列表按 level 分组展示
- [ ] 验证应用级插件路由生成正确（`/apps/{appId}/...`）
- [ ] 查看数据库日志，确保无约束违反或 NULL 相关错误

## 回滚计划（如需）

若迁移失败需要回滚：

```sql
-- 1. 删除新列（数据丢失风险，仅在紧急情况下执行）
ALTER TABLE sk_plugins DROP COLUMN plugin_level;
ALTER TABLE sk_plugins DROP COLUMN app_id;
ALTER TABLE sk_plugins DROP COLUMN mount_policy;

-- 2. 恢复服务
-- - 停止当前后端
-- - 切换回旧版本代码（未应用 P2 改动）
-- - 重启后端
```

## 特殊场景处理

### 场景 1：历史清单中已有 `ui_mode` 但无 `level`

**处理方式**：保持默认 level=system，由加载器自动补齐。

### 场景 2：需要精确标识某插件为应用级

**方法**：在插件清单中显式声明 `level: app` 和 `app_id`，重新安装或手动 UPDATE 数据库。

### 场景 3：应用级插件迁移到系统级（或反之）

**方法**：
```sql
UPDATE sk_plugins 
SET plugin_level = 'system', app_id = NULL 
WHERE id = 'some-plugin-id';
```

## 故障排查

### 问题：迁移后列显示 NULL

**原因**：SQL 步骤 2 未正确执行或条件不匹配。

**解决**：
```sql
SELECT COUNT(*) FROM sk_plugins WHERE plugin_level IS NULL;
-- 若有结果，执行：
UPDATE sk_plugins SET plugin_level = 'system', mount_policy = 'admin' WHERE plugin_level IS NULL;
```

### 问题：后端启动报错 "unknown column"

**原因**：代码已更新但数据库迁移未执行。

**解决**：
1. 执行 SQL 步骤 1（添加新列）
2. 重启后端

### 问题：前端无法加载应用级插件

**原因**：路由路径推导逻辑与后端不一致，或 app_id 为空。

**解决**：
1. 查看后端返回的 frontendEntry 是否正确
2. 检查 level 和 appId 字段是否在响应中
3. 执行 `go test ./internal/handler/http/v1/plugin/...` 验证

## 相关文件清单

| 文件 | 变化 | PR/分支 |
| --- | --- | --- |
| `internal/plugin/types.go` | +level/appId/mountPolicy 字段与验证 | P2 Phase 1 |
| `internal/plugin/loader.go` | +清单解析与自动推导 | P2 Phase 1 |
| `internal/store/sql/gormrepo/plugin_model.go` | +ORM 映射 | P2 Phase 1 |
| `internal/handler/http/v1/plugin/handler.go` | +API 响应字段 | P2 Phase 1 |
| `internal/bootstrap/di.go` | +内置插件默认元数据 | P2 Phase 1 |
| `web/src/plugins/types.ts` | +类型定义 | P2 Phase 2 |
| `web/src/plugins/index.ts` | +路由推导改进 | P2 Phase 2 |
| `web/src/views/Plugin/index.vue` | +分组展示 | P2 Phase 2 |
| `web/src/stores/plugins.ts` | +分组 getter | P2 Phase 2 |

## 参考

- [插件化架构设计](refactor.md#10-插件化架构设计)
- [插件清单契约](refactor.md#104-插件清单契约兼容演进)
- [兼容演进策略](refactor.md#101-兼容策略)
