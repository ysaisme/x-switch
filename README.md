# mswitch

Model API hot-switch proxy — 在多个 LLM API 供应商之间无缝热切换。

## 特性

- **热切换** — 一键切换 API 供应商，下游工具零感知
- **多协议支持** — OpenAI / Anthropic / Gemini，统一 OpenAI 兼容格式对外
- **SSE 流式代理** — 完整支持 streaming 响应透传
- **故障自动切换** — 健康检查 + fallback 路由 + 自动恢复
- **安全控制** — API Key AES-256-GCM 加密、Token 鉴权、IP 白名单、速率限制
- **请求日志** — SQLite 存储，支持查询与统计
- **费用追踪** — 自动估算费用，支持余额查询与 Webhook 告警
- **Web 管理界面** — 内嵌 React UI，开箱即用
- **单文件部署** — 无外部依赖，一个二进制搞定

## 快速开始

```bash
# 安装
go install github.com/user/mswitch/cmd/mswitch@latest

# 或从源码构建
git clone https://github.com/user/mswitch.git
cd mswitch
make build-all    # 构建前端 + Go 二进制

# 初始化
mswitch init

# 启动
mswitch start

# 查看状态
mswitch status

# 热切换
mswitch use site openai-official
mswitch use model gpt-4o azure-eastus
mswitch use profile production
```

## 配置

配置文件位于 `~/.mswitch/config.yaml`：

```yaml
proxy:
  listen: ":9090"        # 代理监听地址
  web_listen: ":9091"    # Web UI 监听地址

sites:
  - id: openai-official
    name: OpenAI
    base_url: https://api.openai.com
    protocol: openai
    api_key: sk-xxx
    models: [gpt-4o, gpt-4o-mini]

  - id: anthropic-direct
    name: Anthropic
    base_url: https://api.anthropic.com
    protocol: anthropic
    api_key: sk-ant-xxx
    models: [claude-sonnet-4-20250514]

  - id: gemini-direct
    name: Gemini
    base_url: https://generativelanguage.googleapis.com
    protocol: gemini
    api_key: xxx
    models: [gemini-2.0-flash]

routing:
  active_profile: default
  profiles:
    - name: default
      rules:
        - model_pattern: "gpt-*"
          site: openai-official
        - model_pattern: "claude-*"
          site: anthropic-direct
        - model_pattern: "*"
          site: openai-official
          fallback: anthropic-direct

security:
  access_token: "your-secret-token"
  allowed_ips: []
  rate_limit:
    global_rpm: 100
    per_site_rpm: 30

logging:
  enabled: true
  max_days: 30
```

## 使用方式

将你的 AI 工具（Cursor、Continue 等）API 地址指向 mswitch 代理：

```
API Base URL: http://localhost:9090/v1
API Key: 你的实际 API Key（或配置 access_token）
```

mswitch 会根据当前活跃 profile 的路由规则，将请求转发到正确的 API 供应商。

## CLI 命令

```
mswitch init              初始化配置
mswitch start             启动代理服务
mswitch stop              停止代理服务
mswitch status            查看运行状态
mswitch current           查看当前路由
mswitch use <profile>     切换到指定 profile
mswitch use site <id>     所有请求路由到指定站点
mswitch use model <m> <s> 指定模型路由到指定站点
mswitch site list         列出所有站点
mswitch site add          添加站点
mswitch site test <id>    测试站点连通性
mswitch profile list      列出所有 profile
mswitch profile create    创建 profile
mswitch balance [site]    查看余额
mswitch logs              查看请求日志
mswitch config edit       编辑配置文件
mswitch config show       显示当前配置
```

## Web UI

启动后访问 `http://localhost:9091` 即可使用 Web 管理界面：

- **仪表盘** — 站点概览、当前路由
- **切换中心** — 一键切换站点/Profile
- **站点管理** — 添加/查看站点
- **请求日志** — 实时查看请求记录
- **用量统计** — Token 用量与费用
- **设置** — 配置重载

## 架构

```
Client (Cursor/Continue/etc.)
    ↓ http://localhost:9090
┌──────────────────────┐
│     mswitch proxy    │
│  ┌────────────────┐  │
│  │  Auth + Rate   │  │
│  │   Limit MW     │  │
│  └───────┬────────┘  │
│          ↓           │
│  ┌────────────────┐  │
│  │     Router     │  │
│  │ (profile/rule) │  │
│  └───────┬────────┘  │
│          ↓           │
│  ┌────────────────┐  │
│  │   Adapter      │  │
│  │ OpenAI/Anth/   │  │
│  │   Gemini       │  │
│  └───────┬────────┘  │
│          ↓           │
│  ┌────────────────┐  │
│  │  Failover +    │  │
│  │  Balance +     │  │
│  │  Logger        │  │
│  └────────────────┘  │
└──────────────────────┘
    ↓           ↓           ↓
  OpenAI     Anthropic    Gemini
```

## 构建

```bash
make build-all    # 构建前端 + 二进制
make build        # 仅构建 Go 二进制（需先 build-web）
make build-web    # 仅构建前端
make release      # 跨平台构建
make clean        # 清理
```

## License

MIT
