# troll

Bilibili 视频评论分析工具。抓取视频评论数据存入本地数据库，提供 TUI 交互界面和 CLI 命令行查询。

## 功能

- 按关键词搜索话题，批量抓取话题下所有视频的评论（含子评论）
- 按 BVid / AVid 抓取单个视频评论
- TUI 交互界面：浏览话题、视频、评论，搜索，查看热门用户，相似评论检测，标记用户管理，数据概览
- CLI 查询：热门用户排行、相似/重复评论、关键词搜索、用户评论历史
- 多账号 Cookie 轮转，自动限流与惩罚/恢复机制
- 扫码登录添加账号
- 支持 SQLite 和 PostgreSQL

## 技术栈

| 层 | 技术 |
|---|------|
| CLI | urfave/cli/v3 |
| TUI | charmbracelet/bubbletea + lipgloss |
| ORM | GORM（SQLite / PostgreSQL） |
| 限流 | golang.org/x/time |

## 构建

```bash
go build -o troll .
```

## 使用

```bash
# 启动 TUI 交互界面
./troll

# 抓取评论
./troll fetch -t <话题关键词>
./troll fetch -b <bvid>
./troll fetch -a <avid>

# 查询
./troll query --top user          # 热门用户
./troll query --top comment       # 相似评论
./troll query --user <uid>        # 用户评论历史
./troll query --keyword <关键词>   # 关键词搜索

# 账号管理
./troll config login              # 扫码登录
./troll config --list             # 列出账号
./troll config --clean            # 清理无效账号
./troll config --proxy <url>      # 设置代理
```

全局参数：`-T` 指定话题/目录名，`-I` 设置请求间隔（默认 2 秒）。

## 配置

`config.yaml`：

```yaml
auth:
  accounts:
    - cookie: "<bilibili cookie>"
      refresh_token: "<refresh token>"
cache_ttl_hours: 24
database:
  name: bliveDB
  type: sqlite    # 或 postgres
```
