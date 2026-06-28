# SQLite → PostgreSQL 数据迁移工具

将 bdanmu / troll 的 SQLite 数据库迁移到 PostgreSQL。

## 构建

```bash
cd tools/migrate
go build -o migrate.exe .
```

## 用法

### 仅清空 PostgreSQL 表

```bash
migrate.exe -pg "host=localhost user=postgres password=123456 dbname=bilibili port=5432 sslmode=disable TimeZone=Asia/Shanghai" --clean
```

### 迁移数据（清空后导入）

```bash
migrate.exe -sqlite .\data\db\bliveDB.db -pg "host=localhost user=postgres password=123456 dbname=bilibili port=5432 sslmode=disable TimeZone=Asia/Shanghai" --clean
```

### 迁移数据（保留已有数据，跳过重复）

```bash
migrate.exe -sqlite .\data\db\bliveDB.db -pg "host=localhost user=postgres password=123456 dbname=bilibili port=5432 sslmode=disable TimeZone=Asia/Shanghai"
```

## 参数

| 参数 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `-pg` | 是 | - | PostgreSQL 连接字符串 |
| `-sqlite` | 否 | - | SQLite 文件路径。省略时仅执行 `--clean` 操作 |
| `--clean` | 否 | false | 迁移前清空所有 PostgreSQL 表 |
| `-batch` | 否 | 500 | 批量插入大小 |

## 迁移的表

| 序号 | 表 | 说明 |
|------|------|------|
| 1 | user_tables | 用户 |
| 2 | user_history_name_tables | 用户历史昵称 |
| 3 | live_room_tables | 直播间 |
| 4 | video_tables | 视频 |
| 5 | medal_tables | 粉丝勋章 |
| 6 | dan_mu_tables | 弹幕 |
| 7 | comment_tables | 评论 |
| 8 | signed_user_tables | 标记用户 |
| 9 | user_entry_tables | 用户进入记录 |
| 10 | configuration_tables | 配置 |

## 注意事项

- 迁移期间自动禁用外键约束，完成后恢复
- 字符串中的 null 字节（`\x00`）会被自动过滤
- 重复主键记录会被跳过（`ON CONFLICT DO NOTHING`）
- PostgreSQL 数据库需提前创建好，工具会自动建表
