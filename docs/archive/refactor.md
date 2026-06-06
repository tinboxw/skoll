# Skoll 项目重构计划方案

> 版本: v1.0\
> 日期: 2026-05-09\
> 作者: Skoll Framework Team\
> Go Module: `github.com/tinboxw/skoll`

***

## 目录

1. [项目背景与目标](#1-项目背景与目标)
2. [技术选型](#2-技术选型)
3. [目录结构设计](#3-目录结构设计)
   - 3.1 [目录总览](#31-目录总览)
   - 3.2 [插件架构方案对比](#32-插件架构方案对比)
   - 3.3 [目录职责详解](#33-目录职责详解)
4. [分阶段实施计划](#4-分阶段实施计划)
5. [开发规范](#5-开发规范)
6. [测试策略](#6-测试策略)
7. [文档体系规划](#7-文档体系规划)
8. [风险评估与应对策略](#8-风险评估与应对策略)
9. [验收标准](#9-验收标准)
10. [插件化架构设计](#10-插件化架构设计)
11. [前端后台管理界面](#11-前端后台管理界面)
12. [插件管理](#12-插件管理)
13. [插件系统开发指南](#13-插件系统开发指南)
14. [前端后台管理系统功能实现指南](#14-前端后台管理系统功能实现指南)
15. [插件打包整合方案](#15-插件打包整合方案)

***

## 1. 项目背景与目标

### 1.1 项目背景

Skoll 是一个基于 Go 语言的开源后台框架，旨在提供高性能、可扩展的企业级管理后台解决方案。本计划基于以下核心参考：

| 参考项目          | 参考维度      | 优先级 |
| ------------- | --------- | --- |
| Go-Admin      | 核心架构思路    | 基础  |
| Gin-Vue-Admin | 接口契约、功能设计 | 高   |
| HisiPHP       | 成熟业务功能经验  | 中   |

### 1.2 核心目标

| 目标类别     | 具体目标    | 量化指标      |
| -------- | ------- | --------- |
| **性能**   | 高并发支持   | QPS ≥ 500 |
| **性能**   | API响应时间 | < 150ms   |
| **可维护性** | 代码覆盖率   | ≥ 80%     |
| **可维护性** | 圈复杂度    | ≤ 10      |
| **扩展性**  | 新增存储后端  | < 1周      |
| **扩展性**  | 新增业务模块  | < 2周      |

### 1.3 架构原则

1. **依赖倒置原则**：高层模块依赖抽象，不依赖具体实现
2. **单一职责原则**：每个模块/组件只负责一个功能领域
3. **开闭原则**：对扩展开放，对修改关闭
4. **接口隔离原则**：细粒度接口定义，避免不必要的依赖

***

## 2. 技术选型

### 2.1 核心技术栈

| 分类          | 技术            | 版本    | 选型依据              | 用途          |
| ----------- | ------------- | ----- | ----------------- | ----------- |
| 语言          | Go            | 1.22+ | 高性能、并发模型、生态成熟     | 核心开发语言      |
| Web框架       | Gin           | 1.9+  | 高性能路由、中间件丰富、社区活跃  | HTTP服务      |
| ORM         | GORM          | 1.25+ | 成熟稳定、支持多数据库、API友好 | 关系型数据库操作    |
| SQL Builder | goqu          | 9.0+  | 类型安全、灵活的SQL构建     | 复杂查询场景      |
| 配置管理        | Viper         | 1.18+ | 多格式支持、热更新、环境变量集成  | 配置管理        |
| 日志          | Zap           | 1.27+ | 高性能结构化日志、低延迟      | 日志记录        |
| 监控          | Prometheus    | 2.47+ | 成熟监控体系、多维指标       | 指标采集        |
| 认证          | JWT           | 标准    | 无状态认证、跨平台兼容       | 用户认证        |
| **主数据库**    | MySQL         | 8.0+  | 成熟稳定、生态完善、读写性能优异  | 核心业务数据存储    |
| **分析数据库**   | PostgreSQL    | 16+   | 高级特性支持、JSONB、全文搜索 | 复杂查询、数据分析   |
| **时序数据库**   | ClickHouse    | 24+   | 列式存储、高性能分析、实时查询   | 日志分析、时序数据   |
| **分布式缓存**   | Redis         | 7.0+  | 高性能KV存储、数据结构丰富    | 会话管理、热点数据缓存 |
| **缓存层**     | Memcached     | 1.6+  | 轻量级缓存、高并发读写       | 页面缓存、临时数据   |
| 消息队列        | Redis Pub/Sub | 7.0+  | 轻量级消息传递           | 事件通知、异步处理   |

### 2.2 存储技术架构

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           应用层                                       │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        ▼                           ▼                           ▼
┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
│   Redis         │       │   Memcached     │       │   本地缓存       │
│  (热点数据)     │       │  (页面缓存)     │       │  (进程内缓存)    │
└─────────────────┘       └─────────────────┘       └─────────────────┘
        │                           │
        └───────────────────────────┼─────────────────────────────┐
                                    ▼                             ▼
                         ┌─────────────────┐             ┌─────────────────┐
                         │    MySQL        │             │   PostgreSQL    │
                         │  (主业务存储)    │             │  (分析存储)     │
                         └─────────────────┘             └─────────────────┘
                                    │
                                    ▼
                         ┌─────────────────┐
                         │   ClickHouse    │
                         │  (时序/日志)     │
                         └─────────────────┘
```

### 2.3 技术栈决策矩阵

| 决策维度        | 选择         | 理由                      |
| ----------- | ---------- | ----------------------- |
| Web框架       | Gin        | 比Echo性能更优，中间件生态更成熟      |
| ORM         | GORM       | 比XORM文档更完善，社区更活跃，支持多数据库 |
| SQL Builder | goqu       | 类型安全、灵活构建复杂SQL          |
| 配置          | Viper      | 支持多格式、热更新、环境变量集成        |
| 日志          | Zap        | 比Logrus性能更高，结构化更好       |
| 主数据库        | MySQL      | 成熟稳定，生态完善，适合OLTP场景      |
| 分析数据库       | PostgreSQL | 高级特性丰富，适合复杂查询和OLAP      |
| 时序数据库       | ClickHouse | 列式存储，高性能实时分析            |
| 分布式缓存       | Redis      | 数据结构丰富，支持Pub/Sub        |
| 轻量缓存        | Memcached  | 低延迟、高并发，适合简单KV场景        |

***

## 3. 目录结构设计

### 3.1 整体架构

```
skoll/                                    # 项目根目录
├── cmd/                                   # 命令入口层
│   └── skoll/                             # 主命令
│       ├── main.go                        # 应用入口
│       ├── server.go                      # 服务器启动逻辑
│       └── shutdown.go                    # 优雅关闭处理
├── internal/                              # 内部模块（不对外暴露）
│   ├── bootstrap/                         # 启动引导层
│   │   ├── config.go                      # 配置加载
│   │   ├── di.go                          # 依赖注入容器
│   │   ├── middleware.go                  # 全局中间件
│   │   └── validator.go                   # 请求校验器
│   ├── domain/                            # 领域层（核心业务实体）
│   │   ├── user/                          # 用户领域
│   │   │   ├── entity.go                  # 用户实体定义
│   │   │   ├── value_object.go            # 值对象（如Email、Password）
│   │   │   └── rules.go                   # 业务规则
│   │   ├── role/                          # 角色领域
│   │   │   ├── entity.go
│   │   │   └── rules.go
│   │   ├── rbac/                          # 权限领域
│   │   │   ├── entity.go
│   │   │   ├── policy.go                  # 策略规则
│   │   │   └── data_scope.go              # 数据范围定义
│   │   ├── audit/                         # 审计领域
│   │   │   └── entity.go
│   │   ├── system/                        # 系统领域
│   │   │   └── entity.go
│   │   └── shared/                        # 共享领域组件
│   │       └── types.go                   # 通用类型定义
│   ├── repository/                        # 仓储层（数据访问契约）
│   │   ├── user_repo.go                   # 用户仓储接口
│   │   ├── role_repo.go                   # 角色仓储接口
│   │   ├── rbac_repo.go                   # 权限仓储接口
│   │   ├── audit_repo.go                  # 审计仓储接口
│   │   ├── system_repo.go                 # 系统仓储接口
│   │   └── transaction.go                 # 事务接口定义
│   ├── store/                             # 存储实现层
│   │   ├── memory/                        # 内存存储（开发/测试环境）
│   │   │   ├── user_store.go
│   │   │   ├── role_store.go
│   │   │   └── rbac_store.go
│   │   ├── sql/                           # SQL存储抽象
│   │   │   ├── mysql/                     # MySQL实现
│   │   │   │   ├── user_store.go
│   │   │   │   ├── role_store.go
│   │   │   │   ├── rbac_store.go
│   │   │   │   └── adapter.go             # MySQL适配器
│   │   │   ├── postgres/                  # PostgreSQL实现
│   │   │   │   ├── user_store.go
│   │   │   │   ├── role_store.go
│   │   │   │   └── adapter.go             # PostgreSQL适配器
│   │   │   ├── common.go                  # SQL通用工具
│   │   │   └── transaction.go             # SQL事务实现
│   │   ├── clickhouse/                    # ClickHouse时序存储
│   │   │   ├── audit_store.go             # 审计日志存储
│   │   │   ├── metrics_store.go           # 指标存储
│   │   │   └── adapter.go                 # ClickHouse适配器
│   │   └── factory.go                     # 存储工厂
│   ├── cache/                             # 缓存层
│   │   ├── redis/                         # Redis缓存
│   │   │   ├── session_cache.go           # 会话缓存
│   │   │   ├── user_cache.go              # 用户缓存
│   │   │   ├── permission_cache.go        # 权限缓存
│   │   │   └── adapter.go                 # Redis适配器
│   │   ├── memcached/                     # Memcached缓存
│   │   │   ├── page_cache.go              # 页面缓存
│   │   │   ├── config_cache.go            # 配置缓存
│   │   │   └── adapter.go                 # Memcached适配器
│   │   ├── local/                         # 本地缓存（进程内）
│   │   │   └── lru_cache.go               # LRU缓存实现
│   │   └── factory.go                     # 缓存工厂
│   ├── service/                           # 业务服务层
│   │   ├── user/                          # 用户服务
│   │   │   ├── service.go                 # 服务接口
│   │   │   ├── service_impl.go            # 服务实现
│   │   │   ├── types.go                   # 请求/响应类型
│   │   │   └── validator.go               # 业务校验
│   │   ├── role/                          # 角色服务
│   │   │   ├── service.go
│   │   │   ├── service_impl.go
│   │   │   └── types.go
│   │   ├── rbac/                          # 权限服务
│   │   │   ├── service.go
│   │   │   ├── service_impl.go
│   │   │   └── types.go
│   │   ├── audit/                         # 审计服务
│   │   │   ├── service.go
│   │   │   └── service_impl.go
│   │   └── common/                        # 通用服务组件
│   │       └── transaction_manager.go     # 事务管理器
│   ├── handler/                           # 接口处理层
│   │   ├── http/                          # HTTP处理器
│   │   │   ├── v1/                        # API v1版本
│   │   │   │   ├── user_handler.go        # 用户API
│   │   │   │   ├── role_handler.go        # 角色API
│   │   │   │   └── rbac_handler.go        # 权限API
│   │   │   ├── router.go                  # 路由注册
│   │   │   └── response.go                # 统一响应封装
│   │   ├── cli/                           # CLI命令
│   │   │   ├── user_cmd.go
│   │   │   └── system_cmd.go
│   │   └── middleware/                    # 中间件
│   │       ├── auth.go                    # 认证中间件
│   │       ├── logger.go                  # 日志中间件
│   │       └── rate_limit.go              # 限流中间件
│   ├── event/                             # 事件总线
│   │   ├── bus.go                         # 事件总线接口
│   │   ├── publisher.go                   # 事件发布者
│   │   ├── subscriber.go                  # 事件订阅者
│   │   └── events/                        # 事件定义
│   │       └── user_events.go
│   └── plugin/                            # 插件层（插件化架构核心）
│       ├── manager.go                     # 插件管理器
│       ├── registry.go                    # 扩展点注册中心
│       ├── loader.go                      # 插件加载器
│       ├── resolver.go                    # 依赖解析器
│       ├── permission.go                  # 插件权限控制
│       ├── types.go                       # 插件类型定义
│       └── builtin/                       # 内置插件目录
│           ├── auth/                      # 认证插件
│           ├── logger/                    # 日志插件
│           └── dashboard/                 # 仪表盘插件
├── plugins/                               # 第三方插件安装目录（运行时生成）
│   └── demo/                              # 插件ID命名的目录
│       ├── plugin.yaml                    # 插件元数据配置
│       ├── main.go                        # 插件主入口
│       ├── handler.go                     # 插件处理器
│       └── static/                        # 插件静态资源
├── web/                                   # 前端后台管理界面
│   ├── src/
│   │   ├── main.ts                        # 入口文件
│   │   ├── App.vue                        # 根组件
│   │   ├── router/                        # 路由配置
│   │   │   └── index.ts                   # 路由定义
│   │   ├── stores/                        # Pinia状态管理
│   │   │   ├── user.ts                    # 用户状态
│   │   │   ├── app.ts                     # 应用状态
│   │   │   └── plugins.ts                 # 插件状态
│   │   ├── components/                    # 公共组件
│   │   │   ├── Layout/                    # 布局组件
│   │   │   │   ├── Sidebar.vue            # 侧边栏
│   │   │   │   ├── Header.vue             # 顶部导航
│   │   │   │   └── MainContent.vue        # 主内容区
│   │   │   └── Common/                    # 通用组件
│   │   │       ├── Table.vue              # 表格组件
│   │   │       ├── Form.vue               # 表单组件
│   │   │       └── Dialog.vue             # 弹窗组件
│   │   ├── views/                         # 页面视图
│   │   │   ├── Dashboard/                 # 仪表盘
│   │   │   │   └── index.vue
│   │   │   ├── User/                      # 用户管理
│   │   │   │   ├── list.vue
│   │   │   │   ├── add.vue
│   │   │   │   └── edit.vue
│   │   │   ├── Role/                      # 角色管理
│   │   │   │   ├── list.vue
│   │   │   │   └── edit.vue
│   │   │   ├── Permission/                # 权限管理
│   │   │   │   └── index.vue
│   │   │   ├── Plugin/                    # 插件管理
│   │   │   └── Setting/                   # 系统设置
│   │   │       └── index.vue
│   │   ├── plugins/                       # 前端插件目录
│   │   │   ├── index.ts                   # 插件注册
│   │   │   └── builtin/                   # 内置插件
│   │   ├── utils/                         # 工具函数
│   │   │   ├── api.ts                     # API封装
│   │   │   ├── auth.ts                    # 认证工具
│   │   │   └── common.ts                  # 通用工具
│   │   └── styles/                        # 样式文件
│   │       ├── global.scss                # 全局样式
│   │       └── variables.scss             # 样式变量
│   ├── public/                            # 静态资源
│   ├── index.html                         # HTML模板
│   ├── package.json                       # 依赖配置
│   ├── vite.config.ts                     # Vite配置
│   └── tsconfig.json                      # TypeScript配置
├── pkg/                                   # 公共工具包（对外暴露）
│   ├── config/                            # 配置管理
│   │   ├── loader.go                      # 配置加载器
│   │   ├── validator.go                   # 配置校验
│   │   └── watcher.go                     # 配置热更新
│   ├── logging/                           # 日志框架
│   │   ├── logger.go                      # 日志接口
│   │   ├── zap_adapter.go                 # Zap适配器
│   │   └── formatter.go                   # 日志格式化
│   ├── metrics/                           # 监控指标
│   │   ├── collector.go                   # 指标采集器
│   │   ├── exporter.go                    # 指标导出器
│   │   └── prometheus_adapter.go          # Prometheus适配器
│   ├── errors/                            # 错误处理
│   │   ├── error.go                       # 错误类型定义
│   │   ├── handler.go                     # 错误处理器
│   │   └── translator.go                  # 错误信息翻译
│   ├── security/                          # 安全工具
│   │   ├── jwt.go                         # JWT工具
│   │   ├── password.go                    # 密码哈希
│   │   └── encrypt.go                     # 加密工具
│   ├── validator/                         # 参数校验
│   │   ├── validator.go                   # 校验器接口
│   │   └── rules.go                       # 校验规则
│   └── utils/                             # 通用工具
│       ├── string.go                      # 字符串工具
│       ├── time.go                        # 时间工具
│       └── convert.go                     # 类型转换
├── migrations/                            # 数据库迁移脚本
│   ├── mysql/                             # MySQL迁移
│   │   ├── 20240101_000001_create_users.sql
│   │   └── 20240102_000002_create_roles.sql
│   └── postgres/                          # PostgreSQL迁移
│       └── 20240101_000001_create_users.sql
├── docs/                                  # 文档
│   ├── api/                               # API文档
│   ├── development/                       # 开发文档
│   ├── user/                              # 用户文档
│   └── architecture/                      # 架构文档
├── tests/                                 # 测试目录
│   ├── unit/                              # 单元测试
│   ├── integration/                       # 集成测试
│   └── benchmark/                         # 性能测试
├── deploy/                                # 部署配置
│   ├── docker/                            # Docker配置
│   │   └── Dockerfile
│   ├── k8s/                               # Kubernetes配置
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   └── compose/                           # Docker Compose配置
│       └── docker-compose.yaml
├── go.mod                                 # Go模块依赖
├── go.sum                                 # 依赖校验和
└── README.md                              # 项目说明
```

### 3.2 插件架构方案对比

#### 3.2.1 四种架构方案目录结构

**方案零：仅前端插件**

```
myplugin/
├── plugin.yaml           # 插件元数据
└── static/               # 前端静态资源（或独立前端构建产物）
  ├── index.html
  ├── app.js
  └── style.css
```

**方案一：仅后端插件**

```
myplugin/
├── plugin.yaml           # 插件元数据
├── go.mod               # Go模块定义
├── main.go              # 插件主入口
├── handler.go           # HTTP处理器
└── service.go           # 业务逻辑
```

**方案二：前后端不分离**

```
myplugin/
├── plugin.yaml           # 插件元数据
├── go.mod               # Go模块定义
├── main.go              # 插件主入口
├── handler.go           # HTTP处理器
├── service.go           # 业务逻辑
└── static/              # 前端静态资源（直接放置构建产物）
    ├── index.html
    ├── app.js
    └── style.css
```

**方案三：前后端分离（推荐）**

```
myplugin/
├── plugin.yaml           # 插件元数据
├── backend/             # 后端代码
│   ├── go.mod
│   ├── main.go
│   ├── handler.go
│   └── service.go
└── frontend/            # 前端代码
    ├── package.json
    ├── src/
    │   ├── App.vue
    │   └── main.ts
    ├── vite.config.ts
    └── dist/            # 构建产物（打包时生成）
```

#### 3.2.2 方案对比分析

| 特性        | 仅前端插件         | 仅后端插件        | 前后端不分离    | 前后端分离     |
| --------- | ------------- | ------------ | --------- | --------- |
| **适用场景**  | 展示页、前端扩展入口    | 纯API服务、无UI需求 | 简单页面、快速开发 | 复杂界面、团队协作 |
| **开发体验**  | 前端技术栈单独维护     | 单一技术栈        | 前后端耦合     | 独立开发、并行迭代 |
| **构建复杂度** | 低到中（取决于前端构建） | 低            | 中         | 高         |
| **代码复用**  | 前端侧复用较高       | 有限           | 有限        | 高         |
| **部署灵活性** | 高             | 高            | 中         | 高         |
| **性能优化**  | 可独立前端优化       | N/A          | 受限        | 充分优化空间    |
| **团队协作**  | 前端团队友好        | 简单           | 受限        | 支持前后端独立团队 |

#### 3.2.3 方案选择建议

| 场景       | 推荐方案   | 理由            |
| -------- | ------ | ------------- |
| 前端展示扩展   | 仅前端插件  | 不依赖后端逻辑、上线快   |
| 数据处理插件   | 仅后端插件  | 无需前端界面        |
| 简单工具页面   | 前后端不分离 | 快速开发、部署简单     |
| 复杂管理后台   | 前后端分离  | 独立开发、性能优化     |
| 大型团队协作项目 | 前后端分离  | 职责清晰、并行开发能力强 |

### 3.3 目录职责详解

#### 3.3.1 命令入口层 (`cmd/`)

| 目录/文件         | 职责    | 说明             |
| ------------- | ----- | -------------- |
| `cmd/skoll/`  | 应用主入口 | 包含main.go和启动逻辑 |
| `main.go`     | 程序入口  | 初始化并启动应用       |
| `server.go`   | 服务器配置 | HTTP服务器初始化     |
| `shutdown.go` | 优雅关闭  | 处理信号、释放资源      |

#### 3.2.2 启动引导层 (`internal/bootstrap/`)

| 文件              | 职责    | 说明           |
| --------------- | ----- | ------------ |
| `config.go`     | 配置加载  | 从文件/环境变量加载配置 |
| `di.go`         | 依赖注入  | 管理组件依赖关系     |
| `middleware.go` | 全局中间件 | 注册跨域、日志等中间件  |
| `validator.go`  | 请求校验  | 统一参数校验入口     |

#### 3.2.3 领域层 (`internal/domain/`)

| 目录               | 职责   | 说明            |
| ---------------- | ---- | ------------- |
| `domain/user/`   | 用户领域 | 用户实体、值对象、业务规则 |
| `domain/role/`   | 角色领域 | 角色实体、权限关联     |
| `domain/rbac/`   | 权限领域 | 策略规则、数据范围     |
| `domain/audit/`  | 审计领域 | 审计记录实体        |
| `domain/system/` | 系统领域 | 系统配置、参数       |
| `domain/shared/` | 共享组件 | 通用类型定义        |

#### 3.2.4 仓储层 (`internal/repository/`)

| 文件               | 职责     | 说明         |
| ---------------- | ------ | ---------- |
| `user_repo.go`   | 用户仓储接口 | 定义用户数据访问契约 |
| `role_repo.go`   | 角色仓储接口 | 定义角色数据访问契约 |
| `rbac_repo.go`   | 权限仓储接口 | 定义权限数据访问契约 |
| `audit_repo.go`  | 审计仓储接口 | 定义审计数据访问契约 |
| `transaction.go` | 事务接口   | 定义事务操作契约   |

#### 3.2.5 存储实现层 (`internal/store/`)

| 目录                    | 职责           | 说明         |
| --------------------- | ------------ | ---------- |
| `store/memory/`       | 内存存储         | 开发/测试环境使用  |
| `store/sql/mysql/`    | MySQL实现      | 主业务数据存储    |
| `store/sql/postgres/` | PostgreSQL实现 | 分析型数据存储    |
| `store/clickhouse/`   | ClickHouse实现 | 时序数据、日志存储  |
| `store/factory.go`    | 存储工厂         | 根据配置创建存储实例 |

#### 3.2.6 缓存层 (`internal/cache/`)

| 目录                 | 职责          | 说明         |
| ------------------ | ----------- | ---------- |
| `cache/redis/`     | Redis缓存     | 会话、热点数据缓存  |
| `cache/memcached/` | Memcached缓存 | 页面、配置缓存    |
| `cache/local/`     | 本地缓存        | 进程内LRU缓存   |
| `cache/factory.go` | 缓存工厂        | 根据配置创建缓存实例 |

#### 3.2.7 业务服务层 (`internal/service/`)

| 目录                | 职责   | 说明       |
| ----------------- | ---- | -------- |
| `service/user/`   | 用户服务 | 用户业务逻辑编排 |
| `service/role/`   | 角色服务 | 角色业务逻辑编排 |
| `service/rbac/`   | 权限服务 | 权限业务逻辑编排 |
| `service/audit/`  | 审计服务 | 审计日志记录   |
| `service/common/` | 通用组件 | 事务管理等    |

#### 3.2.8 接口处理层 (`internal/handler/`)

| 目录                    | 职责          | 说明          |
| --------------------- | ----------- | ----------- |
| `handler/http/v1/`    | HTTP API v1 | RESTful接口实现 |
| `handler/cli/`        | CLI命令       | 命令行工具       |
| `handler/middleware/` | 中间件         | 认证、日志、限流    |

#### 3.2.9 事件总线 (`internal/event/`)

| 文件              | 职责     | 说明         |
| --------------- | ------ | ---------- |
| `bus.go`        | 事件总线接口 | 定义事件发布订阅机制 |
| `publisher.go`  | 事件发布者  | 发布领域事件     |
| `subscriber.go` | 事件订阅者  | 订阅并处理事件    |

#### 3.2.10 公共工具包 (`pkg/`)

| 目录               | 职责   | 说明           |
| ---------------- | ---- | ------------ |
| `pkg/config/`    | 配置管理 | 配置加载、校验、热更新  |
| `pkg/logging/`   | 日志框架 | 结构化日志        |
| `pkg/metrics/`   | 监控指标 | Prometheus集成 |
| `pkg/errors/`    | 错误处理 | 统一错误类型       |
| `pkg/security/`  | 安全工具 | JWT、密码哈希     |
| `pkg/validator/` | 参数校验 | 通用校验器        |
| `pkg/utils/`     | 通用工具 | 字符串、时间等工具    |

#### 3.2.11 数据库迁移 (`migrations/`)

| 目录                     | 职责             | 说明       |
| ---------------------- | -------------- | -------- |
| `migrations/mysql/`    | MySQL迁移脚本      | 数据库表结构变更 |
| `migrations/postgres/` | PostgreSQL迁移脚本 | 数据库表结构变更 |

#### 3.2.12 部署配置 (`deploy/`)

| 目录                | 职责             | 说明                 |
| ----------------- | -------------- | ------------------ |
| `deploy/docker/`  | Docker配置       | Dockerfile         |
| `deploy/k8s/`     | Kubernetes配置   | Deployment、Service |
| `deploy/compose/` | Docker Compose | 本地开发环境配置           |

***

## 4. 分阶段实施计划

### 4.1 阶段总览

| 阶段     | 周期   | 核心目标          | 交付物                                  |
| ------ | ---- | ------------- | ------------------------------------ |
| **M0** | 1-2周 | 项目初始化与基础设施搭建  | 项目骨架、工具链、配置模块                        |
| **M1** | 2-3周 | 核心领域模型与仓储设计   | 领域实体、仓储接口定义                          |
| **M2** | 4-5周 | 存储层实现（多数据库支持） | Memory/MySQL/PostgreSQL/ClickHouse存储 |
| **M3** | 2-3周 | 缓存层实现         | Redis/Memcached/本地缓存                 |
| **M4** | 3-4周 | 业务服务层开发       | 核心业务逻辑、事务管理                          |
| **M5** | 2-3周 | API接口层开发      | RESTful API、中间件、CLI命令                |
| **M6** | 2周   | 事件总线与异步处理     | 事件发布订阅机制                             |
| **M7** | 2周   | 集成测试与优化       | 测试报告、性能基准、文档完善                       |

### 4.2 详细阶段计划

#### M0: 项目初始化与基础设施搭建（第1-2周）

**阶段目标**：完成项目骨架搭建，建立开发规范和工具链

**核心任务**：

| 任务   | 描述               | 负责人  | 依赖   |
| ---- | ---------------- | ---- | ---- |
| M0.1 | Go Module初始化     | 架构师  | 无    |
| M0.2 | 目录结构创建           | 架构师  | M0.1 |
| M0.3 | 配置管理模块（Viper）    | 工程师A | M0.1 |
| M0.4 | 日志模块（Zap）        | 工程师A | M0.3 |
| M0.5 | 错误处理模块           | 工程师B | M0.1 |
| M0.6 | 依赖注入模块           | 架构师  | M0.3 |
| M0.7 | 安全工具模块（JWT/密码哈希） | 工程师B | M0.1 |

**预期成果**：

- `go.mod` / `go.sum`
- `pkg/config/` - 配置管理（加载、校验、热更新）
- `pkg/logging/` - 日志框架（Zap适配器）
- `pkg/errors/` - 错误处理（统一错误类型）
- `pkg/security/` - 安全工具（JWT、密码哈希）
- `internal/bootstrap/di.go` - DI容器

**时间估算**：2周

***

#### M1: 核心领域模型与仓储设计（第3-5周）

**阶段目标**：定义核心业务实体和数据访问契约

**核心任务**：

| 任务   | 描述                | 负责人  | 依赖        |
| ---- | ----------------- | ---- | --------- |
| M1.1 | 用户领域模型（实体、值对象、规则） | 工程师C | M0        |
| M1.2 | 角色领域模型            | 工程师C | M1.1      |
| M1.3 | RBAC权限模型（策略、数据范围） | 工程师D | M1.1      |
| M1.4 | 审计日志模型            | 工程师D | M0        |
| M1.5 | 系统配置模型            | 工程师D | M0        |
| M1.6 | 仓储接口定义（含事务接口）     | 架构师  | M1.1-M1.5 |

**预期成果**：

- `internal/domain/user/` - 用户实体、值对象、业务规则
- `internal/domain/role/` - 角色实体
- `internal/domain/rbac/` - 权限策略、数据范围
- `internal/domain/audit/` - 审计实体
- `internal/domain/system/` - 系统配置实体
- `internal/repository/` - 仓储接口（user/role/rbac/audit/system）

**时间估算**：3周

***

#### M2: 存储层实现（多数据库支持）（第6-10周）

**阶段目标**：实现Memory、MySQL、PostgreSQL、ClickHouse存储适配

**核心任务**：

| 任务   | 描述                 | 负责人  | 依赖        |
| ---- | ------------------ | ---- | --------- |
| M2.1 | Memory存储基础         | 工程师E | M1        |
| M2.2 | 用户/角色/权限Memory存储   | 工程师E | M2.1      |
| M2.3 | SQL存储基础（GORM）      | 工程师F | M1        |
| M2.4 | MySQL适配器与存储实现      | 工程师F | M2.3      |
| M2.5 | PostgreSQL适配器与存储实现 | 工程师G | M2.3      |
| M2.6 | ClickHouse适配器与存储实现 | 工程师H | M1        |
| M2.7 | 存储工厂实现             | 架构师  | M2.4-M2.6 |
| M2.8 | 数据库迁移脚本            | 工程师F | M2.4-M2.5 |

**预期成果**：

- `internal/store/memory/` - 内存存储实现
- `internal/store/sql/mysql/` - MySQL存储实现
- `internal/store/sql/postgres/` - PostgreSQL存储实现
- `internal/store/clickhouse/` - ClickHouse时序存储
- `internal/store/factory.go` - 存储工厂
- `migrations/mysql/` - MySQL迁移脚本
- `migrations/postgres/` - PostgreSQL迁移脚本

**时间估算**：5周

***

#### M3: 缓存层实现（第11-13周）

**阶段目标**：实现Redis、Memcached、本地缓存

**核心任务**：

| 任务   | 描述                | 负责人  | 依赖        |
| ---- | ----------------- | ---- | --------- |
| M3.1 | Redis适配器与缓存实现     | 工程师I | M2        |
| M3.2 | 会话缓存、用户缓存、权限缓存    | 工程师I | M3.1      |
| M3.3 | Memcached适配器与缓存实现 | 工程师J | M2        |
| M3.4 | 页面缓存、配置缓存         | 工程师J | M3.3      |
| M3.5 | 本地LRU缓存实现         | 工程师K | M2        |
| M3.6 | 缓存工厂实现            | 架构师  | M3.1-M3.5 |

**预期成果**：

- `internal/cache/redis/` - Redis缓存实现
- `internal/cache/memcached/` - Memcached缓存实现
- `internal/cache/local/` - 本地LRU缓存
- `internal/cache/factory.go` - 缓存工厂

**时间估算**：3周

***

#### M4: 业务服务层开发（第14-17周）

**阶段目标**：实现核心业务逻辑和事务管理

**核心任务**：

| 任务   | 描述          | 负责人  | 依赖        |
| ---- | ----------- | ---- | --------- |
| M4.1 | 用户服务（接口+实现） | 工程师C | M3        |
| M4.2 | 角色服务        | 工程师C | M3        |
| M4.3 | RBAC权限服务    | 工程师D | M3        |
| M4.4 | 审计服务        | 工程师D | M3        |
| M4.5 | 事务管理器       | 架构师  | M4.1-M4.4 |
| M4.6 | 业务校验器       | 工程师B | M4.1-M4.4 |

**预期成果**：

- `internal/service/user/` - 用户业务逻辑
- `internal/service/role/` - 角色业务逻辑
- `internal/service/rbac/` - 权限业务逻辑
- `internal/service/audit/` - 审计业务逻辑
- `internal/service/common/transaction_manager.go` - 事务管理

**时间估算**：4周

***

#### M5: API接口层开发（第18-20周）

**阶段目标**：实现RESTful API、中间件、CLI命令，预留Gin-Vue-Admin接口契约

**核心任务**：

| 任务   | 描述             | 负责人  | 依赖   |
| ---- | -------------- | ---- | ---- |
| M5.1 | HTTP服务器配置（Gin） | 工程师H | M4   |
| M5.2 | 用户API接口（v1）    | 工程师H | M4.1 |
| M5.3 | 角色API接口（v1）    | 工程师H | M4.2 |
| M5.4 | 权限API接口（v1）    | 工程师I | M4.3 |
| M5.5 | 认证中间件          | 架构师  | M5.1 |
| M5.6 | 日志/限流中间件       | 工程师A | M5.1 |
| M5.7 | CLI命令          | 工程师J | M4   |

**预期成果**：

- `internal/handler/http/v1/` - HTTP API v1
- `internal/handler/http/router.go` - 路由注册
- `internal/handler/http/response.go` - 统一响应封装
- `internal/handler/middleware/` - 中间件（认证、日志、限流）
- `internal/handler/cli/` - CLI命令

**时间估算**：3周

***

#### M6: 事件总线与异步处理（第21-22周）

**阶段目标**：实现事件发布订阅机制

**核心任务**：

| 任务   | 描述              | 负责人  | 依赖   |
| ---- | --------------- | ---- | ---- |
| M6.1 | 事件总线接口定义        | 架构师  | M4   |
| M6.2 | 事件发布者实现         | 工程师K | M6.1 |
| M6.3 | 事件订阅者实现         | 工程师K | M6.1 |
| M6.4 | 核心领域事件定义        | 工程师C | M6.1 |
| M6.5 | Redis Pub/Sub集成 | 工程师I | M6.1 |

**预期成果**：

- `internal/event/bus.go` - 事件总线接口
- `internal/event/publisher.go` - 事件发布者
- `internal/event/subscriber.go` - 事件订阅者
- `internal/event/events/` - 领域事件定义

**时间估算**：2周

***

#### M7: 集成测试与优化（第23-24周）

**阶段目标**：完成测试覆盖和性能优化

**核心任务**：

| 任务   | 描述               | 负责人  | 依赖        |
| ---- | ---------------- | ---- | --------- |
| M7.1 | 单元测试（领域层/仓储层）    | 全员   | M1-M6     |
| M7.2 | 集成测试（服务层/接口层）    | 工程师I | M5        |
| M7.3 | 性能测试（k6/vegeta）  | 工程师I | M5        |
| M7.4 | 代码优化（静态分析）       | 架构师  | M7.1-M7.3 |
| M7.5 | 文档完善             | 全员   | 全部        |
| M7.6 | 部署配置（Docker/K8s） | 工程师J | M5        |

**预期成果**：

- 测试覆盖率 ≥ 80%
- 性能基准报告（QPS ≥ 500，响应时间 < 150ms）
- 代码质量分析报告
- 完整项目文档
- Dockerfile、Kubernetes配置

**时间估算**：2周

***

## 5. 开发规范

### 5.1 代码规范

#### 5.1.1 文件命名

- 使用小写字母和下划线：`user_service.go`
- 避免使用驼峰命名：`userservice.go` ❌

#### 5.1.2 包命名

- 包名简洁，使用小写：`user`, `role`, `rbac`
- 避免使用复数：`users` ❌

#### 5.1.3 类型命名

- 使用 PascalCase：`type UserService struct{}`
- 接口以 `er` 结尾：`type UserRepository interface{}`

#### 5.1.4 变量命名

- 使用 camelCase：`userID`, `roleName`
- 避免使用缩写：`usr` ❌, `user` ✅

#### 5.1.5 注释规范

- 所有公开类型、函数、方法必须有注释
- 注释使用完整句子，以句号结尾
- 包注释放在 `package` 声明之前

### 5.2 提交规范

#### 5.2.1 提交格式

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

#### 5.2.2 类型说明

| 类型         | 说明                  |
| ---------- | ------------------- |
| `feat`     | 新功能                 |
| `fix`      | 修复bug               |
| `docs`     | 文档更新                |
| `style`    | 代码格式（不影响功能）         |
| `refactor` | 重构（既不是新增功能也不是修复bug） |
| `test`     | 测试相关                |
| `chore`    | 构建/工具相关             |

#### 5.2.3 示例

```
feat(user): 添加用户创建接口

- 实现 CreateUser 服务方法
- 添加用户创建 HTTP Handler
- 更新 API 文档

closes #123
```

### 5.3 文档规范

#### 5.3.1 文档类型

| 类型    | 位置                  | 说明               |
| ----- | ------------------- | ---------------- |
| API文档 | `docs/api/`         | RESTful API 接口文档 |
| 开发文档  | `docs/development/` | 架构设计、开发指南        |
| 用户文档  | `docs/user/`        | 使用说明、部署指南        |

#### 5.3.2 格式要求

- 使用 Markdown 格式
- 代码块使用语言标识
- 表格使用标准 Markdown 格式
- 图片使用相对路径

***

## 6. 测试策略

### 6.1 测试类型

| 测试类型  | 覆盖范围    | 工具                  |
| ----- | ------- | ------------------- |
| 单元测试  | 单个函数/方法 | `go test`           |
| 集成测试  | 模块间交互   | `go test` + 测试数据库   |
| 端到端测试 | 完整业务流程  | `go test` + HTTP客户端 |
| 性能测试  | 并发、响应时间 | `vegeta` / `k6`     |

### 6.2 测试覆盖率目标

| 模块          | 覆盖率目标 |
| ----------- | ----- |
| Service层    | ≥ 80% |
| Repository层 | ≥ 70% |
| Handler层    | ≥ 60% |
| Store层      | ≥ 70% |

### 6.3 测试执行策略

```bash
# 运行所有单元测试
go test ./... -v -cover

# 运行特定模块测试
go test ./internal/service/user/... -v

# 生成覆盖率报告
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# 性能测试
go test ./tests/benchmark/... -bench=. -benchmem
```

### 6.4 CI/CD集成

- **CI触发条件**：每次push自动运行单元测试
- **CD触发条件**：合并到main分支自动构建部署
- **质量门禁**：测试覆盖率低于阈值时阻止合并

***

## 7. 文档体系规划

### 7.1 文档目录结构

```
docs/
├── api/                    # API文档
│   ├── user.md             # 用户模块API
│   ├── role.md             # 角色模块API
│   ├── rbac.md             # 权限模块API
│   └── swagger.yaml        # Swagger规范
├── development/            # 开发文档
│   ├── architecture.md     # 架构设计
│   ├── coding_standard.md  # 编码规范
│   ├── database.md         # 数据库设计
│   └── contributing.md     # 贡献指南
├── user/                   # 用户文档
│   ├── quickstart.md       # 快速开始
│   ├── deployment.md       # 部署指南
│   └── configuration.md    # 配置说明
└── README.md               # 项目说明
```

### 7.2 文档产出标准

| 文档类型  | 产出时间 | 更新频率    | 负责人     |
| ----- | ---- | ------- | ------- |
| API文档 | M4阶段 | 每次API变更 | 接口开发者   |
| 架构文档  | M0阶段 | 架构变更时   | 架构师     |
| 开发规范  | M0阶段 | 定期评审    | 技术负责人   |
| 用户文档  | M5阶段 | 功能变更时   | 技术文档工程师 |

### 7.3 文档维护机制

1. **文档即代码**：文档与代码同仓库管理
2. **CI检查**：文档格式检查纳入CI流程
3. **版本同步**：文档版本与代码版本保持一致
4. **定期评审**：每季度进行文档完整性评审

***

## 8. 风险评估与应对策略

### 8.1 技术风险

| 风险          | 概率 | 影响 | 应对策略                  |
| ----------- | -- | -- | --------------------- |
| GORM多数据库兼容性 | 中  | 高  | 抽象SQL层、使用通用接口、编写方言适配层 |
| 事务一致性       | 中  | 高  | 引入分布式事务框架、实现补偿机制      |
| 性能瓶颈        | 低  | 高  | 引入缓存策略、读写分离、连接池调优     |
| 安全漏洞        | 低  | 高  | 定期安全审计、依赖版本监控         |

### 8.2 进度风险

| 风险     | 概率 | 影响 | 应对策略                  |
| ------ | -- | -- | --------------------- |
| 需求变更   | 高  | 中  | 采用敏捷迭代、建立需求冻结机制、优先级评估 |
| 技术难点阻塞 | 中  | 高  | 技术预研先行、引入专家咨询、预留缓冲时间  |
| 人员变动   | 低  | 高  | 代码评审机制、完善文档、知识共享      |

### 8.3 质量风险

| 风险     | 概率 | 影响 | 应对策略                 |
| ------ | -- | -- | -------------------- |
| 测试覆盖不足 | 中  | 高  | CI强制覆盖率检查、代码评审时检查测试  |
| 代码质量下降 | 中  | 中  | 静态代码分析、代码评审规范、技术债务追踪 |
| 文档缺失   | 高  | 中  | 文档即代码、自动化文档生成、定期文档评审 |

### 8.4 应对策略矩阵

| 风险类型 | 预防措施       | 缓解措施      | 应急措施         |
| ---- | ---------- | --------- | ------------ |
| 技术风险 | 技术调研、POC验证 | 架构评审、代码审查 | 快速回滚、临时方案    |
| 进度风险 | 任务拆解、时间估算  | 每日站会、进度追踪 | 资源调配、优先级调整   |
| 质量风险 | 代码规范、测试驱动  | 静态分析、代码评审 | Bug修复流程、回归测试 |

***

## 9. 验收标准

### 9.1 各阶段验收标准

#### M0 验收标准

- [x] Go Module初始化完成
- [x] 目录结构创建完成
- [x] 配置管理模块可用（Viper集成）
- [x] 日志模块可用（Zap集成）
- [x] 错误处理模块可用
- [x] 依赖注入模块可用
- [x] 安全工具模块可用（JWT/密码哈希）

#### M1 验收标准

- [x] 用户领域模型定义完成（实体、值对象、规则）
- [x] 角色领域模型定义完成
- [x] RBAC权限模型定义完成（策略、数据范围）
- [x] 审计日志模型定义完成
- [x] 系统配置模型定义完成
- [x] 仓储接口定义完成（含事务接口）
- [x] 领域单元测试覆盖率 ≥ 70%

#### M2 验收标准

- [x] Memory存储实现完成
- [x] MySQL存储实现完成
- [x] PostgreSQL存储实现完成
- [x] ClickHouse存储实现完成
- [x] 存储工厂实现完成
- [x] 数据库迁移脚本完成（MySQL/PostgreSQL）
- [x] 存储层测试覆盖率 ≥ 70%

#### M3 验收标准

- [x] Redis缓存实现完成
- [x] Memcached缓存实现完成
- [x] 本地LRU缓存实现完成
- [x] 缓存工厂实现完成
- [x] 缓存层测试覆盖率 ≥ 70%

#### M4 验收标准

- [x] 用户服务实现完成（接口+实现）
- [x] 角色服务实现完成
- [x] RBAC权限服务实现完成
- [x] 审计服务实现完成
- [x] 事务管理器实现完成
- [x] 业务校验器实现完成
- [x] 服务层测试覆盖率 ≥ 80%

#### M5 验收标准

- [x] HTTP服务器配置完成（Gin）
- [x] 用户API接口完成（v1）
- [x] 角色API接口完成（v1）
- [x] 权限API接口完成（v1）
- [x] 认证中间件完成
- [x] 日志/限流中间件完成
- [x] CLI命令完成
- [x] Handler层测试覆盖率 ≥ 60%

#### M6 验收标准

- [x] 事件总线接口定义完成
- [x] 事件发布者实现完成
- [x] 事件订阅者实现完成
- [x] 核心领域事件定义完成
- [x] Redis Pub/Sub集成完成

#### M7 验收标准

- [x] 整体测试覆盖率 ≥ 80%
- [x] API响应时间 < 150ms
- [x] QPS ≥ 500
- [x] 圈复杂度 ≤ 10
- [x] 文档完整性 ≥ 90%
- [x] Dockerfile完成
- [x] Kubernetes配置完成

### 9.2 交付物清单

| 阶段 | 交付物                                        | 格式               |
| -- | ------------------------------------------ | ---------------- |
| M0 | 项目骨架                                       | 代码仓库             |
| M0 | 开发规范文档                                     | Markdown         |
| M0 | 配置管理模块                                     | Go源码             |
| M0 | 日志模块                                       | Go源码             |
| M0 | 安全工具模块                                     | Go源码             |
| M1 | 领域模型代码                                     | Go源码             |
| M1 | 仓储接口代码                                     | Go源码             |
| M2 | 存储实现代码（Memory/MySQL/PostgreSQL/ClickHouse） | Go源码             |
| M2 | 数据库迁移脚本                                    | SQL              |
| M3 | 缓存实现代码（Redis/Memcached/Local）              | Go源码             |
| M4 | 服务层代码                                      | Go源码             |
| M5 | API接口代码                                    | Go源码             |
| M5 | API文档                                      | Swagger/Markdown |
| M6 | 事件总线代码                                     | Go源码             |
| M7 | 测试报告                                       | PDF              |
| M7 | 性能基准报告                                     | PDF              |
| M7 | 项目文档                                       | Markdown         |
| M7 | Dockerfile                                 | 配置文件             |
| M7 | Kubernetes配置                               | YAML             |

***

## 附录：甘特图

```
时间轴 (24周)
┌─────────────────────────────────────────────────────────────────────────────────────┐
│ 周数 │ 1 2 │ 3 4 5 │ 6 7 8 9 10 │ 11 12 13 │ 14 15 16 17 │ 18 19 20 │ 21 22 │ 23 24 │
├──────┼──────┼────────┼────────────┼──────────┼─────────────┼──────────┼────────┼────────┤
│ M0   │ ████ │        │            │          │             │          │        │        │
│ M1   │      │ ██████ │            │          │             │          │        │        │
│ M2   │      │        │ ██████████ │          │             │          │        │        │
│ M3   │      │        │            │ ████████ │             │          │        │        │
│ M4   │      │        │            │          │ ██████████  │          │        │        │
│ M5   │      │        │            │          │             │ ████████ │        │        │
│ M6   │      │        │            │          │             │          │ ██████ │        │
│ M7   │      │        │            │          │             │          │        │ ██████ │
└──────┴──────┴────────┴────────────┴──────────┴─────────────┴──────────┴────────┴────────┘
```

***

## 10. 插件化架构设计

### 10.1 设计目标（完整方案）

插件体系从“可安装模块”升级为“可治理的平台能力层”，核心目标：

- 支持系统级插件与应用级插件并存，且具备不同前端访问路径与隔离边界
- 后端、前端、权限、路由、配置、依赖统一由插件元数据驱动
- 保证默认安全（deny-by-default）、可观测（日志/审计）、可演进（兼容旧清单）
- 在项目初期完成基础设施建设，避免后期大规模返工

### 10.2 三层架构

#### 10.2.1 控制面（Control Plane）

负责插件清单解析、安装/启停、依赖校验、策略校验、审计记录。

- 清单入口：`plugin.yaml`
- 生命周期状态：`installed` / `enabled` / `disabled` / `uninstalled`
- 治理能力：版本策略、来源校验、风险分级、灰度启用

#### 10.2.2 运行面（Runtime Plane）

负责路由挂载、扩展点注册、事件分发、资源隔离。

- 后端扩展点：route、middleware、event、permission
- 前端扩展点：menu、dashboard_widget、setting_page、embedded_page
- 路由映射由插件级别与挂载策略共同决定

#### 10.2.3 体验面（Experience Plane）

负责插件管理 UI、插件市场、开发脚手架、诊断工具。

- 管理 API 统一返回插件级别与挂载元数据
- 前端可按插件级别做分组展示与导航编排

### 10.3 插件级别模型（System vs App）

#### 10.3.1 元数据字段

| 字段 | 说明 | 示例 |
| --- | --- | --- |
| `level` | 插件级别，`system` 或 `app` | `system` |
| `app_id` | 应用级插件归属应用（仅 `level=app` 必填） | `crm` |
| `mount_policy` | 前端挂载策略：`admin` / `user` / `mixed` | `admin` |
| `ui_mode` | UI 交付模式（已有） | `separated` |
| `frontend_entry` | 前端入口路径（可省略，由策略推导） | `/plugins/auth` |

#### 10.3.2 路径策略

默认推导规则：

- 系统级插件：`/plugins/{plugin_id}`
- 应用级插件：`/apps/{app_id}/plugins/{plugin_id}`
- 若显式配置 `frontend_entry`，优先使用显式值

说明：页面真实访问前缀由 `SKOLL_WEB_BASE_PATH` 统一注入，插件入口是站内路由路径，不重复硬编码页面基础前缀。

### 10.4 插件清单契约（兼容演进）

```yaml
id: "crm-report"
name: "CRM Report"
version: "1.0.0"
level: "app"
app_id: "crm"
mount_policy: "user"
ui_mode: "frontend_only"
# frontend_entry 可省略，系统将自动推导为 /apps/crm/plugins/crm-report
dependencies:
  - id: "base-auth"
  version: ">=2.0.0"
permissions:
  - "report:read"
  - "report:export"
```

兼容策略：

- 未声明 `level` 时，默认 `system`
- 未声明 `mount_policy` 时，默认 `admin`
- 未声明 `frontend_entry` 时，按级别自动推导

### 10.5 生命周期与治理

#### 10.5.1 生命周期

`安装 -> 校验 -> 注册 -> 启用 -> 运行 -> 停用 -> 卸载`

关键校验：

- Manifest 完整性（id/name/version/ui/level/app_id）
- 依赖与版本约束（SemVer）
- 路径与命名规范（防冲突、防越权）

#### 10.5.2 安全治理

- 权限最小化：插件声明权限仅可申请，不可自动生效
- 路由隔离：系统级与应用级插件路由空间隔离
- 审计留痕：安装/启停/卸载/配置变更均写审计日志
- 系统内置插件保护：不可误卸载（保持现有约束）

### 10.6 目录与模块边界

```
internal/plugin/
├── types.go                 # 插件元模型（level/app_id/mount_policy/ui_mode）
├── loader.go                # 清单解析与兼容策略
├── manager.go               # 生命周期与状态管理
├── registry.go              # 运行时扩展点注册
├── resolver.go              # 依赖与装配解析
├── permission.go            # 插件权限模型
└── builtin/                 # 系统内置插件
```

### 10.7 分阶段落地（开始重构推进）

#### 10.7.1 P1（✅ 已完成）

- 扩展插件元模型：`level`、`app_id`、`mount_policy`
- Loader 支持新字段解析 + 兼容默认值
- 新增前端入口推导策略（按级别自动生成）
- 插件管理 API 对外暴露新字段
- 单元测试补齐（解析、校验、路径推导）

**状态**：✅ 完成（后端+类型+测试）

#### 10.7.2 P2（✅ 已完成）

- 扩展仓储持久化字段与迁移策略
- 前端插件中心按级别分组与过滤
- 应用级插件导航装配（`/apps/{app_id}/...`）
- i18n 国际化支持
- 数据库迁移文档

**状态**：✅ 完成（前端UI+store+分组+迁移方案）

#### 10.7.3 P3（🚀 当前已启动）

**P3a（✅ 已完成）：插件签名校验**
- RSA-SHA256 签名验证实现
- 清单签名字段支持（sign_algo, sign_timestamp, sign_value, vendor_pubkey）
- SignatureChecker 高级 API + 日志
- 数据库序列化（signature_json 字段）
- 完整单元测试（9个测试全部通过）

**P3b（📋 规划中）：插件风险分级**
- 风险评分引擎（来源、权限、UI 模式、维护度）
- 风险等级分类（low/medium/high/restricted）
- 清单风险自评与依赖传递

**P3c（📋 规划中）：策略引擎与决策**
- 组织安全策略定义
- 策略评估与自动决策
- 手动审批流程
- 审计日志与合规报告

**参考文档**：
- [PLUGIN_SECURITY_DESIGN.md](PLUGIN_SECURITY_DESIGN.md) - 完整安全设计
- [PLUGIN_DB_MIGRATION.md](PLUGIN_DB_MIGRATION.md) - 数据库升级

#### 10.7.4 P4（📋 规划中）：扩展与生态

- 租户级策略（tenant-level）隔离
- 插件市场与分发
- 性能与安全基准

***

## 11. 插件管理

### 11.1 插件安装流程与规范

#### 11.1.1 安装流程

```
1. 获取插件包 → 2. 验证插件完整性 → 3. 解析依赖 → 4. 安装依赖 → 5. 复制文件 → 6. 注册插件 → 7. 初始化插件
```

#### 11.1.2 插件包结构规范

```
{plugin-id}/
├── plugin.yaml           # 插件元数据配置（必需）
├── main.go               # 插件主入口（必需）
├── handler.go            # 插件处理器（可选）
├── static/               # 静态资源目录（可选）
├── templates/            # 模板文件目录（可选）
└── config/               # 配置文件目录（可选）
```

#### 11.1.3 `plugin.yaml` 配置规范

```yaml
# 基本信息
id: "example-plugin"              # 插件唯一标识（必需）
name: "示例插件"                   # 插件名称（必需）
version: "1.0.0"                  # 版本号（必需，遵循SemVer）
description: "这是一个示例插件"     # 插件描述（可选）
author: "Author Name"             # 作者信息（可选）
homepage: "https://example.com"   # 插件主页（可选）

# 级别与挂载策略（P1+）
level: "system"                   # 级别：system | app（默认 system）
app_id: "crm"                     # 应用 ID（level=app 时必需）
mount_policy: "admin"             # 挂载策略：admin | user | mixed（默认 admin）

# 供应链安全（P3a+）
vendor: "example-corp"            # 厂商标识（可选）
vendor_url: "https://example.com" # 厂商主页（可选）
sign_algo: "RSA-SHA256"           # 签名算法（可选）
sign_timestamp: "2025-05-18T10:30:00Z"  # 签名时间（可选，RFC3339 格式）
sign_value: "base64-encoded-signature"  # 签名值（可选）
vendor_pubkey: "base64-encoded-pubkey"  # 公钥（PEM PKIX 格式，base64 编码）

# 风险自评（P3b+）
risk:
  level: "medium"                 # 自评风险等级
  reason: "Requires authentication API access"
  dependencies:
    - id: "auth"
      notes: "Uses JWT token validation"

# 功能声明
dependencies:                     # 依赖声明（可选）
  - id: "base-auth"
    version: ">=2.0.0"
  - id: "logger"
    version: "*"
    
permissions:                      # 所需权限（可选）
  - "user:read"
  - "role:manage"

# UI 配置
ui_mode: "monolith"              # UI 模式：backend_only | frontend_only | monolith | separated
frontend_entry: "/plugins/example/dashboard"  # 前端入口（若无，按 level 自动推导）
```

#### 11.1.4 安装方式

**方式一：通过CLI安装**

```bash
# 从本地文件安装
skoll plugin install ./path/to/plugin.zip

# 从远程URL安装
skoll plugin install https://example.com/plugins/myplugin.zip

# 从插件市场安装
skoll plugin install official/myplugin
```

**方式二：通过API安装**

```http
POST /api/v1/plugins/install
Content-Type: multipart/form-data

{
  "file": <plugin.zip>,
  "activate": true
}
```

**方式三：手动安装**

```bash
# 将插件目录复制到插件安装目录
cp -r myplugin/ /path/to/skoll/plugins/

# 重启服务或热加载
skoll plugin reload
```

### 11.2 插件启用与禁用

#### 11.2.1 状态流转

```
installed → [enable] → enabled
enabled   → [disable] → disabled
disabled  → [enable] → enabled
installed → [uninstall] → uninstalled
enabled   → [uninstall] → uninstalled (自动先禁用)
```

#### 11.2.2 操作命令

```bash
# 启用插件
skoll plugin enable <plugin-id>

# 禁用插件
skoll plugin disable <plugin-id>

# 批量操作
skoll plugin enable plugin1 plugin2 plugin3
skoll plugin disable --all
```

#### 11.2.3 API接口

| 接口                              | 方法   | 描述   |
| ------------------------------- | ---- | ---- |
| `/api/v1/plugins/{id}/enable`   | POST | 启用插件 |
| `/api/v1/plugins/{id}/disable`  | POST | 禁用插件 |
| `/api/v1/plugins/batch/enable`  | POST | 批量启用 |
| `/api/v1/plugins/batch/disable` | POST | 批量禁用 |

### 11.3 插件更新与卸载

#### 11.3.1 更新机制

**自动更新检测**：

```bash
# 检查更新
skoll plugin update --check

# 更新指定插件
skoll plugin update <plugin-id>

# 更新所有插件
skoll plugin update --all
```

**更新策略**：

1. 语义化版本检查（SemVer）
2. 差异更新（仅更新变化的文件）
3. 回滚支持（更新失败自动回滚到上一版本）

#### 11.3.2 卸载机制

**卸载流程**：

```
1. 检查插件状态 → 2. 禁用插件（如已启用） → 3. 调用Destroy() → 4. 清理文件 → 5. 更新数据库记录
```

**卸载命令**：

```bash
# 卸载插件（保留配置）
skoll plugin uninstall <plugin-id>

# 卸载插件（删除所有相关数据）
skoll plugin uninstall <plugin-id> --purge
```

**数据清理策略**：

- `--purge`：删除插件所有数据（配置、日志、数据库记录）
- 默认：仅删除插件文件，保留配置备份

### 11.4 插件配置管理

#### 11.4.1 配置层级

```
全局配置 < 插件默认配置 < 用户自定义配置
```

#### 11.4.2 配置文件位置

| 配置类型    | 路径                                        | 说明       |
| ------- | ----------------------------------------- | -------- |
| 插件默认配置  | `plugins/{plugin-id}/config/default.yaml` | 插件自带默认配置 |
| 用户自定义配置 | `config/plugins/{plugin-id}.yaml`         | 用户覆盖配置   |
| 运行时配置   | 数据库 `plugin_config` 表                     | 动态配置     |

#### 11.4.3 配置管理API

| 接口                                  | 方法   | 描述      |
| ----------------------------------- | ---- | ------- |
| `/api/v1/plugins/{id}/config`       | GET  | 获取插件配置  |
| `/api/v1/plugins/{id}/config`       | PUT  | 更新插件配置  |
| `/api/v1/plugins/{id}/config/reset` | POST | 重置为默认配置 |

#### 11.4.4 配置热更新

插件配置支持热更新，无需重启服务：

```go
// 配置变更监听
config.Watch("plugins.myplugin", func(oldVal, newVal interface{}) {
    // 配置变更处理逻辑
})
```

### 11.5 第三方插件与自定义插件

#### 11.5.1 区别对比

| 特性    | 第三方插件      | 自定义插件  |
| ----- | ---------- | ------ |
| 来源    | 插件市场/外部开发者 | 项目内部开发 |
| 维护者   | 第三方开发者     | 项目团队   |
| 安全性   | 需审核验证      | 自主可控   |
| 更新频率  | 依赖外部发布     | 自主控制   |
| 定制化程度 | 有限         | 完全定制   |
| 支持保障  | 社区支持       | 团队支持   |

#### 11.5.2 管理策略

**第三方插件管理**：

1. **审核机制**：安装前进行安全性扫描
2. **沙箱隔离**：限制插件权限范围
3. **版本锁定**：固定使用的插件版本
4. **白名单机制**：仅允许信任的插件来源

**自定义插件管理**：

1. **代码审查**：纳入项目代码审查流程
2. **版本控制**：与主项目一同版本管理
3. **测试覆盖**：要求达到项目测试标准
4. **文档规范**：遵循项目文档规范

#### 11.5.3 插件目录位置说明

**相对路径**：

- 内置插件：`internal/plugin/builtin/{plugin-id}/`
- 第三方插件：`plugins/{plugin-id}/`
- 前端插件：`web/src/plugins/{plugin-id}/`

**绝对路径示例**：

- Linux/macOS：`/opt/skoll/plugins/myplugin/`
- Windows：`C:\skoll\plugins\myplugin\`
- 开发环境：`$GOPATH/src/github.com/skoll/plugins/myplugin/`

#### 11.5.4 插件存储结构

```
plugins/                                    # 插件根目录（运行时自动创建）
├── .index.json                             # 插件索引文件（自动生成）
├── {plugin-id}/                            # 插件目录（以插件ID命名）
│   ├── plugin.yaml                         # 插件元数据
│   ├── main.go                             # 插件主入口
│   ├── handler.go                          # 请求处理器
│   ├── static/                             # 静态资源
│   │   ├── js/                             # JavaScript文件
│   │   ├── css/                            # 样式文件
│   │   └── images/                         # 图片资源
│   ├── config/                             # 配置目录
│   │   └── default.yaml                    # 默认配置
│   └── version.json                        # 版本信息（自动生成）
└── cache/                                  # 插件缓存目录
    └── {plugin-id}/                        # 插件缓存
```

***

## 12. 前端后台管理界面（Vue实现）

### 12.1 前端技术栈

| 分类   | 技术           | 版本    | 用途      |
| ---- | ------------ | ----- | ------- |
| 框架   | Vue          | 3.4+  | 前端框架    |
| UI组件 | Element Plus | 2.6+  | UI组件库   |
| 路由   | Vue Router   | 4.3+  | 路由管理    |
| 状态管理 | Pinia        | 2.1+  | 状态管理    |
| 构建工具 | Vite         | 5.2+  | 构建工具    |
| 样式   | SCSS         | -     | CSS预处理器 |
| 图标   | Lucide Vue   | -next | 图标库     |

### 11.2 前端目录结构

```
web/                              # 前端项目根目录
├── src/
│   ├── main.ts                   # 入口文件
│   ├── App.vue                   # 根组件
│   ├── router/                   # 路由配置
│   │   └── index.ts              # 路由定义
│   ├── stores/                   # Pinia状态管理
│   │   ├── user.ts               # 用户状态
│   │   ├── app.ts                # 应用状态
│   │   └── plugins.ts            # 插件状态
│   ├── components/               # 公共组件
│   │   ├── Layout/               # 布局组件
│   │   │   ├── Sidebar.vue       # 侧边栏
│   │   │   ├── Header.vue        # 顶部导航
│   │   │   └── MainContent.vue   # 主内容区
│   │   └── Common/               # 通用组件
│   │       ├── Table.vue         # 表格组件
│   │       ├── Form.vue          # 表单组件
│   │       └── Dialog.vue        # 弹窗组件
│   ├── views/                    # 页面视图
│   │   ├── Dashboard/            # 仪表盘
│   │   │   └── index.vue
│   │   ├── User/                 # 用户管理
│   │   │   ├── list.vue
│   │   │   ├── add.vue
│   │   │   └── edit.vue
│   │   ├── Role/                 # 角色管理
│   │   │   ├── list.vue
│   │   │   └── edit.vue
│   │   ├── Permission/           # 权限管理
│   │   │   └── index.vue
│   │   ├── Plugin/               # 插件管理
│   │   ├── Setting/              # 系统设置
│   │   │   └── index.vue
│   ├── plugins/                  # 前端插件目录
│   │   ├── index.ts              # 插件注册
│   │   └── builtin/              # 内置插件
│   ├── utils/                    # 工具函数
│   │   ├── api.ts                # API封装
│   │   ├── auth.ts               # 认证工具
│   │   └── common.ts             # 通用工具
│   └── styles/                   # 样式文件
│       ├── global.scss           # 全局样式
│       └── variables.scss        # 样式变量
├── public/                       # 静态资源
├── index.html                    # HTML模板
├── package.json                  # 依赖配置
├── vite.config.ts                # Vite配置
└── tsconfig.json                 # TypeScript配置
```

### 11.3 前端插件机制

#### 11.3.1 前端插件接口

```typescript
interface FrontendPlugin {
  id: string
  name: string
  version: string
  init(app: App): void
  routes?: RouteRecordRaw[]
  components?: Record<string, Component>
  menus?: MenuItem[]
  hooks?: PluginHooks
}
```

#### 11.3.2 前端插件通信

**与后端通信方式**：

- RESTful API调用
- WebSocket实时通信
- 插件专用API端点

**前端插件间通信**：

- 事件总线（Event Bus）
- 共享状态（Pinia）
- 组件插槽（Slot）

### 11.4 页面功能规划

| 页面   | 功能            | 路径            |
| ---- | ------------- | ------------- |
| 仪表盘  | 数据概览、统计图表     | `/dashboard`  |
| 用户管理 | 用户列表、新增、编辑、删除 | `/user`       |
| 角色管理 | 角色列表、权限配置     | `/role`       |
| 权限管理 | 权限策略、数据范围     | `/permission` |
| 插件管理 | 插件列表、安装、启用、禁用 | `/plugin`     |
| 系统设置 | 系统配置、参数管理     | `/setting`    |

### 11.5 前后端插件协同

```
┌─────────────────────────────────────────────────────────────────┐
│                      前端插件层                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                     │
│  │ 插件A UI  │  │ 插件B UI  │  │ 插件C UI  │                    │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘                     │
│       │             │             │                            │
│       └─────────────┼─────────────┘                            │
│                     ▼                                          │
│              前端插件管理器                                      │
│                     │                                          │
└─────────────────────┼───────────────────────────────────────────┘
                      │ HTTP/WebSocket
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                      后端插件层                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                     │
│  │ 插件A API │  │ 插件B API │  │ 插件C API │                    │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘                     │
│       │             │             │                            │
│       └─────────────┼─────────────┘                            │
│                     ▼                                          │
│              后端插件管理器                                      │
│                     │                                          │
└─────────────────────┼───────────────────────────────────────────┘
                      │
                      ▼
              核心框架服务
```

### 11.6 构建与部署

**开发模式**：

```bash
cd web
npm install
npm run dev
```

**生产构建**：

```bash
npm run build
```

***

## 12. 新增阶段计划

### 12.1 插件化架构实施阶段

| 阶段      | 周期 | 核心目标    | 交付物            |
| ------- | -- | ------- | -------------- |
| **M8**  | 3周 | 插件化架构基础 | 插件管理器、加载器、依赖解析 |
| **M9**  | 3周 | 插件扩展点实现 | 路由/中间件/事件扩展点   |
| **M10** | 3周 | 前端插件系统  | 前端插件框架、通信机制    |
| **M11** | 2周 | 插件开发工具  | 调试工具、示例插件、文档   |

### 12.2 更新后的甘特图

```
时间轴 (35周)
┌─────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ 周数 │ 1 2 │ 3 4 5 │ 6-10 │ 11-13 │ 14-17 │ 18-20 │ 21-22 │ 23-24 │ 25-27 │ 28-30 │ 31-33 │ 34-35 │
├──────┼──────┼────────┼──────┼───────┼───────┼───────┼───────┼───────┼───────┼───────┼───────┼───────┤
│ M0   │ ████ │        │      │       │       │       │       │       │       │       │       │       │
│ M1   │      │ ██████ │      │       │       │       │       │       │       │       │       │       │
│ M2   │      │        │ ████ │       │       │       │       │       │       │       │       │       │
│ M3   │      │        │      │ █████ │       │       │       │       │       │       │       │       │
│ M4   │      │        │      │       │ █████ │       │       │       │       │       │       │       │
│ M5   │      │        │      │       │       │ █████ │       │       │       │       │       │       │
│ M6   │      │        │      │       │       │       │ █████ │       │       │       │       │       │
│ M7   │      │        │      │       │       │       │       │ █████ │       │       │       │       │
│ M8   │      │        │      │       │       │       │       │       │ █████ │       │       │       │
│ M9   │      │        │      │       │       │       │       │       │       │ █████ │       │       │
│ M10  │      │        │      │       │       │       │       │       │       │       │ █████ │       │
│ M11  │      │        │      │       │       │       │       │       │       │       │       │ ████ │
└──────┴──────┴────────┴──────┴───────┴───────┴───────┴───────┴───────┴───────┴───────┴───────┴───────┘
```

***

## 13. 插件系统开发指南

### 13.1 插件完整生命周期管理

#### 13.1.1 生命周期阶段

```
开发 → 打包 → 发布 → 安装 → 启用 → 运行 → 禁用 → 卸载
```

#### 13.1.2 开发规范

**代码结构**：

```
myplugin/
├── plugin.yaml           # 必需：插件元数据
├── go.mod                # Go模块定义
├── go.sum                # 依赖校验和
├── main.go               # 必需：插件主入口
├── handler.go            # 可选：HTTP处理器
├── service.go            # 可选：业务逻辑
├── model.go              # 可选：数据模型
├── config.go             # 可选：配置定义
├── event.go              # 可选：事件处理
└── static/               # 可选：前端静态资源
    ├── index.html
    ├── app.js
    └── style.css
```

**命名规范**：

- 插件ID：小写字母+连字符，如 `my-awesome-plugin`
- 文件命名：小写字母+下划线，如 `user_handler.go`
- 包名：与插件ID一致，如 `package myawesomeplugin`

#### 13.1.3 打包流程

**步骤1：验证插件**

```bash
skoll plugin validate ./myplugin/
```

**步骤2：构建插件**

```bash
cd myplugin
go build -buildmode=plugin -o myplugin.so .
```

**步骤3：打包插件**

```bash
zip -r myplugin.zip myplugin.so plugin.yaml static/
```

**打包格式**：

- 支持格式：`.zip`, `.tar.gz`
- 必需包含：`plugin.yaml`, 编译后的插件文件
- 可选包含：`static/`, `config/`

#### 13.1.4 安装步骤

**开发环境安装**：

```bash
# 方式1：直接从源码安装（开发模式）
skoll plugin install ./myplugin/ --dev

# 方式2：从打包文件安装
skoll plugin install ./myplugin.zip

# 方式3：从远程仓库安装
skoll plugin install github.com/user/myplugin@v1.0.0
```

**生产环境安装**：

```bash
# 使用插件市场
skoll plugin install official/myplugin

# 或通过API
curl -X POST http://localhost:8080/api/v1/plugins/install \
  -F "file=@myplugin.zip" \
  -F "activate=true"
```

#### 13.1.5 卸载方法

```bash
# 卸载但保留配置
skoll plugin uninstall myplugin

# 完全卸载（删除所有数据）
skoll plugin uninstall myplugin --purge

# 通过API卸载
curl -X DELETE http://localhost:8080/api/v1/plugins/myplugin?purge=true
```

#### 13.1.6 使用说明

**配置插件**：

```yaml
# config/plugins/myplugin.yaml
enabled: true
api_key: "your-api-key"
timeout: 30s
```

**调用插件API**：

```bash
curl http://localhost:8080/api/v1/myplugin/endpoint
```

### 13.2 插件代码存放位置

#### 13.2.1 开发方式选择

| 开发方式     | 适用场景         | 存放位置                                     | 优点          | 缺点          |
| -------- | ------------ | ---------------------------------------- | ----------- | ----------- |
| **独立项目** | 第三方插件、需要独立发布 | `$GOPATH/src/github.com/user/myplugin`   | 独立版本控制、易于分发 | 需要单独维护、调试复杂 |
| **集成开发** | 内置插件、紧密耦合功能  | `internal/plugin/builtin/myplugin/`      | 便于调试、共享代码   | 发布需跟随主项目    |
| **混合开发** | 大型插件、前后端分离   | 后端：独立项目 / 前端：`web/src/plugins/myplugin/` | 灵活组合        | 协调复杂度高      |

#### 13.2.2 推荐开发流程

```
1. 创建独立插件项目
2. 在主项目中通过 go.mod 引用
3. 开发测试完成后编译打包
4. 通过插件市场或手动安装部署
```

### 13.3 插件接口定义标准

#### 13.3.1 必须实现的接口

**核心接口**：

```go
type Plugin interface {
    // 插件标识
    ID() string
    Name() string
    Version() string
    
    // 生命周期
    Init(ctx context.Context, app *App) error
    Destroy(ctx context.Context) error
    
    // 扩展点注册
    Register(registry ExtensionRegistry) error
}
```

**可选接口**：

```go
// HTTP路由注册
type RouteProvider interface {
    Routes() []RouteDefinition
}

// API端点注册
type APIProvider interface {
    APIs() []APIEndpoint
}

// 事件订阅
type EventSubscriber interface {
    Events() []EventListener
}

// 配置定义
type Configurable interface {
    DefaultConfig() interface{}
    ValidateConfig(config interface{}) error
}
```

#### 13.3.2 数据交换格式

**请求/响应格式**：

```json
// 请求
{
  "method": "POST",
  "path": "/api/v1/myplugin/action",
  "headers": {
    "Content-Type": "application/json"
  },
  "body": {
    "param1": "value1",
    "param2": 123
  }
}

// 响应
{
  "code": 200,
  "message": "success",
  "data": { ... },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

**错误响应格式**：

```json
{
  "code": 400,
  "message": "参数错误",
  "errors": [
    {
      "field": "email",
      "message": "邮箱格式不正确"
    }
  ],
  "timestamp": "2024-01-01T00:00:00Z"
}
```

#### 13.3.3 事件处理机制

**事件定义**：

```go
type PluginEvent struct {
    Name    string
    Payload interface{}
    Context context.Context
}
```

**事件订阅**：

```go
func (p *MyPlugin) Events() []EventListener {
    return []EventListener{
        {
            EventName: "user.created",
            Handler:   p.onUserCreated,
        },
        {
            EventName: "role.updated",
            Handler:   p.onRoleUpdated,
        },
    }
}

func (p *MyPlugin) onUserCreated(ctx context.Context, event *PluginEvent) error {
    // 处理用户创建事件
    user := event.Payload.(user.User)
    // ...
    return nil
}
```

**事件发布**：

```go
err := app.EventBus.Publish(ctx, &PluginEvent{
    Name:    "myplugin.action.completed",
    Payload: result,
})
```

### 13.4 前端实现要求

#### 13.4.1 技术栈要求

**必须使用**：

- Vue 3.x
- Element Plus UI组件库
- TypeScript

**推荐使用**：

- Pinia（状态管理）
- Vue Router（路由）
- SCSS（样式）

#### 13.4.2 前端资源结构

```
myplugin/static/
├── index.html           # 插件入口页面
├── main.ts              # 入口文件
├── App.vue              # 根组件
├── router/              # 路由配置
├── components/          # 自定义组件
├── stores/              # 状态管理
├── api/                 # API调用
└── assets/              # 静态资源
    ├── css/
    ├── js/
    └── images/
```

#### 13.4.3 打包整合方案

**开发模式**：

```bash
# 前端开发服务器
cd web
npm run dev

# 插件前端资源通过HTTP访问
http://localhost:5173/plugins/myplugin/
```

**生产模式**：

```bash
# 构建主应用
npm run build

# 插件资源打包到主应用
# 或独立部署到CDN
```

**资源引用方式**：

```vue
<!-- 动态加载插件组件 -->
<template>
  <component :is="pluginComponent" />
</template>

<script setup>
import { defineAsyncComponent } from 'vue'

const pluginComponent = defineAsyncComponent(() => 
  import(`@/plugins/${pluginId}/App.vue`)
)
</script>
```

***

## 14. 前端后台管理系统功能实现指南

### 14.1 后端功能接口清单

#### 14.1.1 用户管理接口

| 接口     | 方法     | 路径                          | 描述       |
| ------ | ------ | --------------------------- | -------- |
| 获取用户列表 | GET    | `/api/v1/users`             | 分页查询用户   |
| 获取用户详情 | GET    | `/api/v1/users/{id}`        | 根据ID获取用户 |
| 创建用户   | POST   | `/api/v1/users`             | 新增用户     |
| 更新用户   | PUT    | `/api/v1/users/{id}`        | 更新用户信息   |
| 删除用户   | DELETE | `/api/v1/users/{id}`        | 删除用户     |
| 批量删除   | DELETE | `/api/v1/users/batch`       | 批量删除用户   |
| 用户状态切换 | POST   | `/api/v1/users/{id}/status` | 启用/禁用用户  |
| 用户角色分配 | POST   | `/api/v1/users/{id}/roles`  | 分配用户角色   |

#### 14.1.2 角色管理接口

| 接口     | 方法     | 路径                               | 描述       |
| ------ | ------ | -------------------------------- | -------- |
| 获取角色列表 | GET    | `/api/v1/roles`                  | 分页查询角色   |
| 获取角色详情 | GET    | `/api/v1/roles/{id}`             | 根据ID获取角色 |
| 创建角色   | POST   | `/api/v1/roles`                  | 新增角色     |
| 更新角色   | PUT    | `/api/v1/roles/{id}`             | 更新角色信息   |
| 删除角色   | DELETE | `/api/v1/roles/{id}`             | 删除角色     |
| 角色权限配置 | POST   | `/api/v1/roles/{id}/permissions` | 配置角色权限   |
| 获取角色用户 | GET    | `/api/v1/roles/{id}/users`       | 获取角色关联用户 |

#### 14.1.3 权限管理接口

| 接口     | 方法     | 路径                         | 描述       |
| ------ | ------ | -------------------------- | -------- |
| 获取权限列表 | GET    | `/api/v1/permissions`      | 查询所有权限   |
| 获取权限详情 | GET    | `/api/v1/permissions/{id}` | 根据ID获取权限 |
| 创建权限   | POST   | `/api/v1/permissions`      | 新增权限     |
| 更新权限   | PUT    | `/api/v1/permissions/{id}` | 更新权限信息   |
| 删除权限   | DELETE | `/api/v1/permissions/{id}` | 删除权限     |
| 获取权限树  | GET    | `/api/v1/permissions/tree` | 获取权限树形结构 |

#### 14.1.4 插件管理接口

| 接口     | 方法     | 路径                             | 描述       |
| ------ | ------ | ------------------------------ | -------- |
| 获取插件列表 | GET    | `/api/v1/plugins`              | 查询所有插件   |
| 获取插件详情 | GET    | `/api/v1/plugins/{id}`         | 根据ID获取插件 |
| 安装插件   | POST   | `/api/v1/plugins/install`      | 安装插件     |
| 启用插件   | POST   | `/api/v1/plugins/{id}/enable`  | 启用插件     |
| 禁用插件   | POST   | `/api/v1/plugins/{id}/disable` | 禁用插件     |
| 卸载插件   | DELETE | `/api/v1/plugins/{id}`         | 卸载插件     |
| 获取插件配置 | GET    | `/api/v1/plugins/{id}/config`  | 获取插件配置   |
| 更新插件配置 | PUT    | `/api/v1/plugins/{id}/config`  | 更新插件配置   |

#### 14.1.5 系统设置接口

| 接口     | 方法   | 路径                       | 描述      |
| ------ | ---- | ------------------------ | ------- |
| 获取系统配置 | GET  | `/api/v1/system/settings`       | 获取所有配置  |
| 获取单个配置 | GET  | `/api/v1/system/settings/{key}` | 根据键获取配置 |
| 更新单个配置 | PUT  | `/api/v1/system/settings/{key}` | 更新单个配置  |
| 重置配置   | POST | `/api/v1/system/settings/reset` | 重置为默认配置 |

#### 14.1.6 审计日志接口

| 接口     | 方法     | 路径                          | 描述       |
| ------ | ------ | --------------------------- | -------- |
| 获取日志列表 | GET    | `/api/v1/audit`             | 分页查询日志   |
| 获取日志详情 | GET    | `/api/v1/audit/{id}`        | 根据ID获取日志 |
| 导出日志   | GET    | `/api/v1/audit/export`      | 导出日志文件   |
| 清理日志   | DELETE | `/api/v1/audit`             | 清理历史日志   |

### 14.2 前端功能实现计划

#### 14.2.1 实现范围说明

**必须实现**：

- 用户管理完整功能
- 角色管理完整功能
- 权限管理完整功能
- 插件管理完整功能
- 系统设置完整功能
- 审计日志查看功能

**建议实现**：

- 仪表盘数据可视化
- 数据导出功能
- 高级搜索/筛选功能
- 操作日志详情

#### 14.2.2 功能实现优先级

| 优先级 | 模块   | 功能               | 时间预估 |
| --- | ---- | ---------------- | ---- |
| P0  | 用户管理 | 列表、新增、编辑、删除      | 2天   |
| P0  | 角色管理 | 列表、新增、编辑、删除、权限配置 | 3天   |
| P0  | 权限管理 | 列表、树形展示、权限分配     | 2天   |
| P1  | 插件管理 | 列表、启用/禁用、配置      | 3天   |
| P1  | 系统设置 | 配置展示、编辑          | 2天   |
| P1  | 审计日志 | 列表、详情            | 2天   |
| P2  | 仪表盘  | 数据统计、图表展示        | 3天   |
| P2  | 通用功能 | 导出、高级搜索          | 2天   |

### 14.3 功能模块划分原则

#### 14.3.1 模块划分策略

**按业务领域划分**：

```
├── 用户管理模块
│   ├── 用户列表
│   ├── 用户详情
│   ├── 用户创建/编辑
│   └── 用户角色管理
├── 角色管理模块
│   ├── 角色列表
│   ├── 角色创建/编辑
│   └── 角色权限配置
├── 权限管理模块
│   ├── 权限列表
│   ├── 权限树形结构
│   └── 权限分配
├── 插件管理模块
│   ├── 插件列表
│   ├── 插件安装
│   └── 插件配置
├── 系统设置模块
│   └── 系统配置管理
├── 审计日志模块
│   └── 日志查看
└── 仪表盘模块
    └── 数据概览
```

#### 14.3.2 界面设计规范

**布局规范**：

- 左侧导航栏：固定宽度 200px
- 顶部导航栏：高度 60px
- 主内容区：自适应剩余空间
- 卡片间距：16px
- 圆角：8px

**色彩规范**：

- 主色调：#1890ff（蓝色）
- 成功色：#52c41a（绿色）
- 警告色：#faad14（橙色）
- 错误色：#f5222d（红色）
- 文本色：#333333（深灰）
- 辅助文本：#666666（中灰）
- 边框色：#e8e8e8（浅灰）

**字体规范**：

- 标题：16px/1.5，粗体
- 正文：14px/1.5，常规
- 辅助文字：12px/1.5，常规
- 按钮文字：14px/1.5，粗体

#### 14.3.3 交互逻辑要求

**通用交互**：

1. **列表页**：分页展示、搜索筛选、批量操作、状态切换
2. **详情页**：信息展示、编辑跳转、相关数据关联展示
3. **表单页**：字段校验、提交反馈、重置功能
4. **弹窗页**：确认提示、加载状态、操作反馈

**数据加载**：

- 初始加载：显示骨架屏
- 列表滚动：无限滚动或分页加载
- 错误处理：友好的错误提示，支持重试

**权限控制**：

- 按钮级权限控制
- 页面级权限控制
- 数据范围权限控制

**操作反馈**：

- 成功：绿色提示，自动消失
- 失败：红色提示，显示详细信息
- 警告：黄色提示，需要确认

***

## 15. 插件打包整合方案

### 15.1 打包前的项目结构规范

#### 15.1.1 标准目录结构（前后端分离架构）

```
myplugin/                                    # 插件根目录
├── plugin.yaml                               # 必需：插件元数据配置
├── backend/                                  # 后端代码目录
│   ├── go.mod                               # Go模块定义
│   ├── go.sum                               # 依赖校验和
│   ├── main.go                              # 插件主入口（实现Plugin接口）
│   ├── handler.go                           # HTTP处理器（注册路由）
│   ├── service.go                           # 业务逻辑层
│   ├── model.go                             # 数据模型定义
│   ├── config.go                            # 配置定义与校验
│   ├── event.go                             # 事件处理
│   └── internal/                            # 内部包（不对外暴露）
│       └── utils.go                         # 工具函数
├── frontend/                                # 前端代码目录
│   ├── package.json                         # 前端依赖配置
│   ├── vite.config.ts                       # Vite构建配置
│   ├── tsconfig.json                        # TypeScript配置
│   ├── index.html                           # HTML入口模板
│   └── src/
│       ├── main.ts                          # 前端入口文件
│       ├── App.vue                          # 根组件
│       ├── router/                          # 路由配置
│       │   └── index.ts
│       ├── stores/                          # Pinia状态管理
│       │   └── index.ts
│       ├── components/                      # 公共组件
│       │   ├── Layout.vue
│       │   └── Table.vue
│       ├── views/                           # 页面视图
│       │   └── MyPluginPage.vue
│       ├── api/                             # API调用封装
│       │   └── index.ts
│       ├── styles/                          # 样式文件
│       │   ├── global.scss
│       │   └── variables.scss
│       └── assets/                          # 静态资源
│           ├── images/                      # 图片文件
│           ├── icons/                       # 图标文件
│           └── fonts/                       # 字体文件
└── scripts/                                 # 构建脚本目录
    ├── build.sh                             # 构建脚本（Linux/macOS）
    ├── build.ps1                            # 构建脚本（Windows）
    └── deploy.sh                            # 部署脚本
```

#### 15.1.2 目录命名规范

| 目录/文件 | 命名规则       | 说明                  |
| ----- | ---------- | ------------------- |
| 根目录   | 小写字母+连字符   | `my-awesome-plugin` |
| Go文件  | 小写字母+下划线   | `user_handler.go`   |
| Vue组件 | PascalCase | `MyPluginPage.vue`  |
| 目录名   | 小写字母+连字符   | `my-plugin-page`    |
| 配置文件  | 小写字母+下划线   | `plugin.yaml`       |

#### 15.1.3 前后端交互接口定义

**API接口规范**：

```yaml
# plugin.yaml中定义API接口
apis:
  - path: "/api/v1/myplugin/data"
    method: "GET"
    description: "获取数据列表"
    request: {}
    response:
      type: "object"
      properties:
        code: { type: "integer" }
        message: { type: "string" }
        data: { type: "array" }
```

**事件接口规范**：

```go
// 后端事件定义
const (
    EventUserCreated = "myplugin.user.created"
    EventDataUpdated = "myplugin.data.updated"
)

// 前端事件监听
eventBus.on('myplugin.user.created', (data) => {
    // 处理事件
})
```

### 15.2 前端资源构建流程

#### 15.2.1 构建工具选型

| 工具             | 用途   | 选型理由                |
| -------------- | ---- | ------------------- |
| **Vite**       | 构建工具 | 快速启动、ES模块支持、Vue官方推荐 |
| **TypeScript** | 类型检查 | 类型安全、更好的开发体验        |
| **Sass/SCSS**  | 样式处理 | 变量、嵌套、Mixin支持       |
| **Rollup**     | 打包工具 | 代码分割、Tree-shaking   |

#### 15.2.2 构建步骤详解

**步骤1：安装依赖**

```bash
cd frontend
npm install
```

**步骤2：源代码编译**

```bash
# TypeScript转JavaScript
npx tsc --noEmit

# SCSS转CSS
npx sass src/styles/:dist/styles/
```

**步骤3：代码压缩与混淆**

```bash
# 生产环境构建
npm run build
```

**步骤4：资源优化**

- 图片压缩：使用`sharp`或`imagemin`
- SVG优化：使用`svgo`
- 字体子集化：仅包含使用的字符

#### 15.2.3 关键配置示例

**vite.config.ts**：

```typescript
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    rollupOptions: {
      input: {
        main: resolve(__dirname, 'index.html')
      }
    },
    chunkSizeWarningLimit: 500,
    cssCodeSplit: true,
    sourcemap: false
  },
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  }
})
```

**package.json脚本**：

```json
{
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview",
    "lint": "eslint . --ext .vue,.js,.jsx,.cjs,.mjs,.ts,.tsx,.cts,.mts --fix --ignore-path .gitignore"
  }
}
```

#### 15.2.4 资源哈希命名配置

```typescript
// vite.config.ts
export default defineConfig({
  build: {
    assetsDir: 'assets',
    rollupOptions: {
      output: {
        assetFileNames: 'assets/[name]-[hash][extname]',
        chunkFileNames: '[name]-[hash].js',
        entryFileNames: '[name]-[hash].js'
      }
    }
  }
})
```

### 15.3 后端代码打包策略

#### 15.3.1 依赖管理

**生产环境依赖筛选**：

```bash
# 清理未使用依赖
go mod tidy

# 仅保留生产依赖
go mod vendor
```

**依赖树分析**：

```bash
# 查看依赖树
go mod graph

# 分析依赖大小
go list -m -json all | jq '.Module, .Version'
```

#### 15.3.2 配置文件环境分离

**目录结构**：

```
backend/
├── config/
│   ├── development.yaml    # 开发环境配置
│   ├── test.yaml           # 测试环境配置
│   └── production.yaml     # 生产环境配置
└── main.go
```

**配置加载逻辑**：

```go
func loadConfig() {
    env := os.Getenv("APP_ENV")
    if env == "" {
        env = "development"
    }
    
    configFile := fmt.Sprintf("config/%s.yaml", env)
    viper.SetConfigFile(configFile)
    viper.ReadInConfig()
}
```

#### 15.3.3 环境变量注入

**注入方式**：

```bash
# 通过.env文件
APP_ENV=production
DATABASE_HOST=localhost
DATABASE_PORT=3306

# 通过命令行参数
skoll plugin install ./myplugin.zip --env=production

# 通过Docker环境变量
docker run -e APP_ENV=production myplugin
```

#### 15.3.4 敏感信息处理

**密钥管理策略**：

1. **环境变量存储**：敏感信息不写入配置文件
2. **配置加密**：使用AES加密敏感配置项
3. **密钥服务**：集成Vault或密钥管理服务

**加密配置示例**：

```go
// 配置加密工具
func encryptConfig(value string, key []byte) (string, error) {
    block, _ := aes.NewCipher(key)
    gcm, _ := cipher.NewGCM(block)
    nonce := make([]byte, gcm.NonceSize())
    io.ReadFull(rand.Reader, nonce)
    return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(value), nil)), nil
}
```

#### 15.3.5 代码编译

**编译命令**：

```bash
# 开发环境
go build -o myplugin.so -buildmode=plugin .

# 生产环境（优化编译）
go build -o myplugin.so -buildmode=plugin -ldflags="-s -w" .

# 跨平台编译
GOOS=linux GOARCH=amd64 go build -o myplugin.so -buildmode=plugin .
```

### 15.4 前后端打包产物整合方法

#### 15.4.1 整合流程

```
1. 前端构建 → 2. 后端编译 → 3. 文件整合 → 4. 打包插件
```

#### 15.4.2 自动化工具选择

| 工具                 | 适用场景    | 优点        | 缺点       |
| ------------------ | ------- | --------- | -------- |
| **npm scripts**    | 简单场景    | 无需额外依赖    | 复杂逻辑难以维护 |
| **Gulp**           | 中等复杂度   | 流式处理、插件丰富 | 学习曲线较陡   |
| **Makefile**       | 复杂构建流程  | 强大灵活、跨平台  | 语法较复杂    |
| **GitHub Actions** | CI/CD集成 | 自动化、可扩展   | 需要学习配置   |

#### 15.4.3 整合脚本示例

**build.sh**：

```bash
#!/bin/bash

# 清理旧构建
rm -rf dist/ myplugin.zip

# 前端构建
cd frontend
npm install
npm run build
cd ..

# 后端编译
cd backend
go build -o myplugin.so -buildmode=plugin .
cd ..

# 创建插件包目录
mkdir -p dist/myplugin

# 复制文件
cp plugin.yaml dist/myplugin/
cp backend/myplugin.so dist/myplugin/
cp -r frontend/dist/* dist/myplugin/static/

# 打包
cd dist
zip -r myplugin.zip myplugin/
cd ..

echo "Build completed: dist/myplugin.zip"
```

**Makefile**：

```makefile
.PHONY: all build frontend backend package clean

all: build package

build: frontend backend

frontend:
	cd frontend && npm install && npm run build

backend:
	cd backend && go build -o myplugin.so -buildmode=plugin .

package:
	mkdir -p dist/myplugin
	cp plugin.yaml dist/myplugin/
	cp backend/myplugin.so dist/myplugin/
	cp -r frontend/dist/* dist/myplugin/static/
	cd dist && zip -r myplugin.zip myplugin/

clean:
	rm -rf dist/ backend/myplugin.so frontend/dist/
```

### 15.5 打包后文件结构说明

#### 15.5.1 最终插件包结构

```
myplugin.zip/                               # 插件包根目录
└── myplugin/                               # 插件目录（与插件ID同名）
    ├── plugin.yaml                         # 必需：插件元数据
    ├── myplugin.so                         # 必需：编译后的Go插件
    ├── static/                             # 前端静态资源
    │   ├── index.html                      # HTML入口
    │   ├── assets/                         # 静态资源目录
    │   │   ├── js/
    │   │   │   ├── app-[hash].js           # 应用主文件
    │   │   │   └── chunk-[hash].js         # 代码分割块
    │   │   ├── css/
    │   │   │   └── app-[hash].css          # 样式文件
    │   │   ├── images/                     # 图片资源
    │   │   └── fonts/                      # 字体资源
    │   └── favicon.ico                     # 图标
    └── config/                             # 可选：默认配置
        └── default.yaml                    # 默认配置文件
```

#### 15.5.2 必需文件说明

**plugin.yaml**：

```yaml
id: "myplugin"
name: "我的插件"
version: "1.0.0"
description: "这是一个示例插件"
author: "Author Name"
dependencies:
  - id: "base-auth"
    version: ">=2.0.0"
permissions:
  - "user:read"
api_version: "1.0"
entry_point: "myplugin.so"
static_dir: "static"
```

#### 15.5.3 插件包校验

```bash
# 验证插件包结构
skoll plugin validate ./myplugin.zip

# 校验内容：
# 1. plugin.yaml是否存在且格式正确
# 2. 入口文件是否存在
# 3. 依赖是否满足
# 4. 目录结构是否符合规范
```

### 15.6 打包过程常见问题及解决方案

#### 15.6.1 资源路径错误

**问题现象**：

```
Error: Failed to load resource: the server responded with a status of 404 (Not Found)
```

**解决方案**：

```typescript
// vite.config.ts
export default defineConfig({
  base: './',  // 设置相对路径
  build: {
    assetsDir: 'assets',
    rollupOptions: {
      output: {
        assetFileNames: 'assets/[name]-[hash][extname]'
      }
    }
  }
})
```

#### 15.6.2 依赖冲突

**问题现象**：

```
Error: found in GOPATH: ... but does not contain package
```

**解决方案**：

```bash
# 清理并重新下载依赖
go clean -modcache
go mod tidy

# 或者使用vendor
go mod vendor
```

#### 15.6.3 构建失败

**问题现象**：

```
error: TS2345: Argument of type 'string | undefined' is not assignable
```

**解决方案**：

```typescript
// 确保类型安全
const value: string = process.env.MY_VAR ?? 'default'
```

#### 15.6.4 环境变量缺失

**问题现象**：

```
panic: runtime error: invalid memory address or nil pointer dereference
```

**解决方案**：

```go
// 添加环境变量校验
func init() {
    if os.Getenv("DATABASE_HOST") == "" {
        log.Fatal("DATABASE_HOST environment variable is required")
    }
}
```

#### 15.6.5 文件权限问题

**问题现象**：

```
Error: permission denied
```

**解决方案**：

```bash
# 设置正确权限
chmod +x build.sh
chmod 755 backend/myplugin.so

# 或在构建时设置
go build -o myplugin.so -buildmode=plugin -mode=0755 .
```

### 15.7 打包命令示例与自动化构建建议

#### 15.7.1 命令行示例

**开发环境构建**：

```bash
# 开发模式（监听文件变化）
npm run dev

# 后端热重载
go build -o myplugin.so -buildmode=plugin . && skoll plugin reload myplugin
```

**生产环境构建**：

```bash
# 完整构建
make all

# 或使用脚本
./scripts/build.sh

# 验证构建结果
skoll plugin validate dist/myplugin.zip
```

**清理命令**：

```bash
# 清理所有构建产物
make clean

# 清理前端构建
rm -rf frontend/dist/

# 清理后端构建
rm backend/myplugin.so
```

#### 15.7.2 CI/CD配置示例

**GitHub Actions**：

```yaml
name: Build Plugin

on:
  push:
    branches: [ main ]

jobs:
  build:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.22'
    
    - name: Set up Node.js
      uses: actions/setup-node@v4
      with:
        node-version: '20'
    
    - name: Build frontend
      run: |
        cd frontend
        npm install
        npm run build
    
    - name: Build backend
      run: |
        cd backend
        go build -o myplugin.so -buildmode=plugin .
    
    - name: Package plugin
      run: make package
    
    - name: Upload artifact
      uses: actions/upload-artifact@v4
      with:
        name: myplugin.zip
        path: dist/myplugin.zip
```

#### 15.7.3 版本号自动管理

**package.json**：

```json
{
  "version": "1.0.0",
  "scripts": {
    "release:patch": "npm version patch && git push && git push --tags",
    "release:minor": "npm version minor && git push && git push --tags",
    "release:major": "npm version major && git push && git push --tags"
  }
}
```

**自动版本注入**：

```go
// main.go
package main

import "fmt"

var version = "1.0.0"

func main() {
    fmt.Printf("Plugin version: %s\n", version)
}
```

```bash
# 编译时注入版本号
go build -ldflags="-X main.version=$(git describe --tags)" -o myplugin.so -buildmode=plugin .
```

***

**文档版本**: v1.3\
**最后更新**: 2026-05-09\
**状态**: 草案

***

*Skoll Framework - 高性能Go后台框架*
