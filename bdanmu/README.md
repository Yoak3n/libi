# bdanmu

Bilibili 直播间弹幕桌面客户端。通过 WebSocket 连接直播间，实时展示弹幕、SC（Super Chat）和用户进入事件。

## 功能

- 扫码登录，支持 Cookie 自动刷新
- 实时弹幕展示，支持表情渲染
- Super Chat 悬浮展示（顶部置顶，自动淡出）
- 用户进入记录追踪
- 用户信息三级缓存（内存 LRU -> SQLite -> Bilibili API）
- 弹幕批量持久化（每 2 秒或达到批量阈值时写入数据库）
- WebSocket 广播服务，可将消息转发给外部客户端
- 系统托盘集成（显示/隐藏窗口、退出）
- 切换直播间

## 技术栈

| 层 | 技术 |
|---|------|
| 后端 | Go, Wails v3, blivedm-go, GORM, coder/websocket |
| 前端 | Vue 3, TypeScript, Vite, Naive UI, Pinia |
| 数据库 | SQLite（默认）/ PostgreSQL |

## 构建

```bash
cd frontend && pnpm install && pnpm build
wails3 build
```

前端产物通过 `//go:embed` 嵌入 Go 二进制文件。

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

PostgreSQL 需额外配置 `host`、`port`、`user`、`password`。
