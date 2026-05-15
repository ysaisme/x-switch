# X Switch

Model API hot-switch proxy — 在多个 LLM API 供应商之间无缝热切换。

## 特性

- **热切换** — 一键切换 API 供应商，下游工具零感知
- **多协议支持** — OpenAI / Anthropic / Gemini，统一 OpenAI 兼容格式对外
- **SSE 流式代理** — 完整支持 streaming 响应透传
- **故障自动切换** — 健康检查 + fallback 路由 + 自动恢复
- **安全控制** — API Key AES-256-GCM 加密、Token 鉴权、IP 白名单、速率限制
- **请求日志** — SQLite 存储，支持查询与统计
- **站点连通性测试** — 一键检测 API 可用性、延迟、可用模型数
- **模型自动发现** — 自动查询站点支持的模型列表，一键保存到配置
- **原生 macOS 应用** — 双击启动，原生窗口 + Dock 图标 + 菜单栏，开箱即用
- **单文件部署** — 无外部依赖，一个二进制搞定
- **深色模式** — 支持浅色/深色/跟随系统，原生窗口标题栏同步适配

## 快速开始

### macOS 应用（推荐）

1. 下载 `xswitch.dmg`
2. 拖拽 `xswitch.app` 到 `/Applications`
3. 双击启动，自动打开管理界面

### 命令行

```bash
# 从源码构建
git clone git@github.com:ysaisme/x-switch.git
cd x-switch
make build-all    # 构建前端 + Go 二进制

# 启动
xswitch start

# 查看状态
xswitch status

# 热切换
xswitch use model gpt-4o azure-eastus
xswitch use profile production
```

## 配置

配置文件位于 `~/.xswitch/config.yaml`：

```yaml
proxy:
  listen: "127.0.0.1:9090"
  web_listen: "127.0.0.1:9091"

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
  api_key_encryption: true
  access_token: ""
  allowed_ips: ["127.0.0.1"]
  rate_limit:
    global_rpm: 60

logging:
  enabled: true
  max_days: 30
  log_body: false
```

## 使用方式

将你的 AI 工具 API 地址指向 X Switch 代理：

```
API Base URL: http://127.0.0.1:9090/v1
API Key: 你的实际 API Key（或配置 access_token）
```

X Switch 会根据当前活跃 profile 的路由规则，将请求转发到正确的 API 供应商。

## CLI 命令

```
xswitch start             启动代理服务
xswitch stop              停止代理服务
xswitch status            查看运行状态
xswitch current           查看当前路由
xswitch use <profile>     切换到指定 profile
xswitch use model <m> <s> 指定模型路由到指定站点
xswitch site list         列出所有站点
xswitch site add          添加站点
xswitch site test <id>    测试站点连通性
xswitch profile list      列出所有 profile
xswitch profile create    创建 profile
xswitch logs              查看请求日志
xswitch config edit       编辑配置文件
xswitch config show       显示当前配置
```

## 管理界面

双击 xswitch.app 启动后自动打开管理界面：

- **仪表盘** — 站点概览、用量统计、当前路由
- **切换中心** — 一键切换 Profile，管理模型路由规则
- **站点管理** — 添加/编辑/删除站点，连通性测试，模型自动发现
- **请求日志** — 实时查看请求记录
- **设置** — 代理/安全/日志配置，外观主题切换

## 架构

```
Client (Codex/OpenCode/Continue/etc.)
    ↓ http://127.0.0.1:9090
┌──────────────────────┐
│   X Switch proxy     │
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
make app          # 构建 macOS .app（原生窗口）
make dmg          # 构建 macOS .dmg 安装包
make release      # 跨平台构建
make clean        # 清理
```

## License

MIT
