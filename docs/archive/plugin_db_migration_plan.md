# 插件管理数据库统一化实施方案

更新时间：2026-05-10
状态：S2 设计基线（可执行）

## 1. 目标

- 插件管理以数据库为单一事实源。
- 插件安装、启停、卸载、配置变更全部可追溯。
- 运行时状态与数据库状态可校验、可修复。

## 2. 表结构设计

### 2.1 sk_plugins

用途：插件主表（元数据与状态）。

字段：
- id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT
- plugin_id VARCHAR(128) NOT NULL UNIQUE
- plugin_type VARCHAR(32) NOT NULL COMMENT 'builtin|upload|link|embed'
- name VARCHAR(128) NOT NULL
- version VARCHAR(64) NOT NULL
- status VARCHAR(32) NOT NULL COMMENT 'installed|enabled|disabled|error'
- source_type VARCHAR(32) NOT NULL COMMENT 'builtin|upload|url|git'
- source_ref VARCHAR(512) NULL
- manifest_json JSON NULL
- config_json JSON NULL
- last_error TEXT NULL
- created_by VARCHAR(128) NULL
- created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
- updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)

索引：
- uk_plugin_id(plugin_id)
- idx_status(status)
- idx_plugin_type(plugin_type)

### 2.2 sk_plugin_releases

用途：插件发布版本与包信息。

字段：
- id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT
- plugin_id VARCHAR(128) NOT NULL
- release_version VARCHAR(64) NOT NULL
- package_name VARCHAR(256) NULL
- package_hash VARCHAR(128) NULL
- package_size BIGINT UNSIGNED NULL
- signature TEXT NULL
- changelog TEXT NULL
- uploaded_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)

索引：
- idx_plugin_version(plugin_id, release_version)

### 2.3 sk_plugin_routes

用途：插件挂载入口（菜单/API/链接/嵌入）。

字段：
- id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT
- plugin_id VARCHAR(128) NOT NULL
- route_type VARCHAR(32) NOT NULL COMMENT 'menu|api|link|embed'
- route_path VARCHAR(512) NOT NULL
- open_mode VARCHAR(32) NULL COMMENT 'same_tab|new_tab|iframe'
- permission_key VARCHAR(128) NULL
- is_enabled TINYINT(1) NOT NULL DEFAULT 1
- created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
- updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)

索引：
- idx_plugin_routes(plugin_id, route_type)

### 2.4 sk_plugin_instances（可选）

用途：多节点运行时状态。

字段：
- id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT
- plugin_id VARCHAR(128) NOT NULL
- node_id VARCHAR(128) NOT NULL
- runtime_status VARCHAR(32) NOT NULL COMMENT 'running|stopped|error'
- last_heartbeat DATETIME(3) NULL
- last_error TEXT NULL
- updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)

索引：
- idx_plugin_node(plugin_id, node_id)

## 3. 迁移策略

阶段 A：建表与回填
1. 新建 sk_plugins / sk_plugin_releases / sk_plugin_routes（可选 sk_plugin_instances）。
2. 启动时扫描内置插件与已安装目录，回填 sk_plugins。
3. 回填插件路由到 sk_plugin_routes。

阶段 B：读路径切换
1. 管理端插件列表改为查询 sk_plugins。
2. 插件详情改为聚合 sk_plugins + sk_plugin_routes + sk_plugin_releases。

阶段 C：写路径切换
1. 启停/卸载/安装改为：先写数据库状态，再由异步任务执行。
2. 任务执行结果回写 status 与 last_error。

阶段 D：一致性校验
1. 定时任务校验运行时与数据库状态。
2. 差异写入告警并自动修复（可配置）。

## 4. 同步机制

- 主数据源：数据库。
- 运行时注册中心：从数据库加载并缓存。
- 同步方式：事件驱动 + 定时巡检。

事件类型：
- plugin.installed
- plugin.enabled
- plugin.disabled
- plugin.uninstalled
- plugin.config.updated

## 5. 与现有系统集成

后端：
- 新增 PluginRepository 与 PluginService。
- PluginManager 启动时读取数据库而非仅依赖本地静态注册。

前端：
- 插件列表/详情/操作 API 统一对接数据库模型。
- 支持按类型筛选（builtin/upload/link/embed）。

## 6. 风险与约束

- 旧插件元数据可能不完整，需要容错回填策略。
- 上线切换阶段必须保留只读回退开关（仅用于应急，不做长期兼容）。
- 大规模插件变更操作建议异步任务化，防止请求超时。

## 7. SQL 草案（MySQL）

```sql
CREATE TABLE IF NOT EXISTS sk_plugins (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  plugin_id VARCHAR(128) NOT NULL,
  plugin_type VARCHAR(32) NOT NULL,
  name VARCHAR(128) NOT NULL,
  version VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  source_type VARCHAR(32) NOT NULL,
  source_ref VARCHAR(512) NULL,
  manifest_json JSON NULL,
  config_json JSON NULL,
  last_error TEXT NULL,
  created_by VARCHAR(128) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uk_plugin_id (plugin_id),
  KEY idx_status (status),
  KEY idx_plugin_type (plugin_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sk_plugin_releases (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  plugin_id VARCHAR(128) NOT NULL,
  release_version VARCHAR(64) NOT NULL,
  package_name VARCHAR(256) NULL,
  package_hash VARCHAR(128) NULL,
  package_size BIGINT UNSIGNED NULL,
  signature TEXT NULL,
  changelog TEXT NULL,
  uploaded_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  KEY idx_plugin_version (plugin_id, release_version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sk_plugin_routes (
  id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  plugin_id VARCHAR(128) NOT NULL,
  route_type VARCHAR(32) NOT NULL,
  route_path VARCHAR(512) NOT NULL,
  open_mode VARCHAR(32) NULL,
  permission_key VARCHAR(128) NULL,
  is_enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  KEY idx_plugin_routes (plugin_id, route_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

## 8. S2 完成标准

- 插件主数据可在数据库中完整查询。
- 启停/安装/卸载均有数据库状态变更记录。
- 管理端插件页完全由数据库驱动。
- 至少 1 条一致性巡检任务可发现并上报差异。
