# shared

公共基础库，为 `bdanmu` 和 `troll` 提供配置管理、数据库访问、数据模型和 HTTP 请求等基础设施。

## 包结构

| 包 | 说明 |
|---|------|
| `config/` | YAML 配置管理（Viper），支持多账号 Cookie 和自动刷新 |
| `database/` | GORM 数据库初始化，支持 SQLite 和 PostgreSQL，自动建表迁移 |
| `domain/model/table/` | GORM 表结构定义（UserTable, VideoTable, CommentTable 等） |
| `domain/model/schema/` | 领域模型（User, DanMu, Room, SuperChat 等），含表结构到领域模型的转换 |
| `repository/interfaces/` | 仓储接口定义（CRUD + 高级查询） |
| `repository/implements/` | GORM 仓储实现 |
| `package/request/` | HTTP 请求工具，自动注入 Cookie、代理和 Bilibili WBI 签名 |
| `login/` | Bilibili 认证：Cookie 校验、刷新、扫码登录 |

## 数据库表

UserTable, UserHistoryNameTable, VideoTable, LiveRoomTable, MedalTable, DanMuTable, CommentTable, SignedUserTable, UserEntryTable, ConfigurationTable
