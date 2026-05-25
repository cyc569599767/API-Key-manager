# KeyManager

KeyManager 是一个本地优先的桌面端 API Key 管理工具，用于集中保存、分组、检索、测试、导入导出和备份还原 API Key 与 Base URL。

项目基于 Wails、Go、Vue 3 和 SQLite 构建，数据默认保存在本机用户配置目录中，API Key 明文不会直接写入数据库。

## 功能特性

- **API Key 统一管理**：保存名称、Provider、环境、所属分组、预算标签、Base URL、测试端点、限流和失败策略。
- **本地加密存储**：API Key 使用本机生成的 `secret.key` 加密后写入 SQLite，列表和详情默认展示脱敏值。
- **可用性检测**：支持单个或批量测试 API Key，记录 HTTP 状态、延迟、模型数量、错误信息、请求和响应摘要。
- **筛选与搜索**：支持按关键词、Provider、状态、环境和所属分组筛选 API Key。
- **批量操作**：支持批量导入、批量删除、批量测试和批量导出 TXT。
- **复制凭据**：可复制 Base URL 与 API Key，用于快速配置客户端或代理工具。
- **审计记录**：记录新增、编辑、禁用、启用、测试、导入导出、备份还原等关键操作。
- **数据备份与还原**：将 `keymanager.sqlite` 和 `secret.key` 打包为 zip，也可从备份 zip 覆盖式还原。
- **桌面应用体验**：使用 Wails 打包为原生桌面应用，前端由 Vue 3 和 Vite 驱动。

## 技术栈

- **桌面框架**：Wails v2
- **后端**：Go
- **前端**：Vue 3、TypeScript、Vite
- **数据库**：SQLite（`modernc.org/sqlite`）
- **数据加密**：AES-GCM，本地 `secret.key` 管理加密密钥

## 项目结构

```text
.
├── app.go                         # Wails 后端入口与前后端绑定方法
├── main.go                        # 应用启动入口
├── wails.json                     # Wails 项目配置
├── go.mod                         # Go 模块依赖
├── build/                         # 应用图标、平台打包配置与构建产物目录
├── frontend/                      # Vue 3 前端项目
│   ├── package.json
│   └── src/
│       ├── App.vue                # 主界面
│       ├── main.ts
│       └── style.css
└── internal/
    ├── backup/                    # 备份、校验、还原逻辑
    ├── crypto/                    # API Key 加密、解密和脱敏
    ├── db/                        # SQLite 打开与迁移
    ├── model/                     # 数据模型与输入输出结构
    ├── repository/                # 数据访问层
    ├── service/                   # 业务逻辑
    ├── storage/                   # 跨平台数据路径
    └── tester/                    # API Key 可用性测试
```

## 环境要求

请先安装以下工具：

- Go 1.25 或更高版本
- Node.js 与 npm
- Wails CLI v2

确认环境：

```bash
go version
node --version
npm --version
wails version
```

## 本地开发

安装前端依赖：

```bash
npm --prefix frontend install
```

启动开发模式：

```bash
wails dev
```

开发模式会启动 Wails 后端和 Vite 前端，并支持前端热更新。

## 构建

构建生产版本：

```bash
wails build
```

Windows 构建完成后，可执行文件通常位于：

```text
build/bin/keymanager.exe
```

只验证前端构建：

```bash
npm --prefix frontend run build
```

运行 Go 测试：

```bash
go test ./...
```

## 数据存储

KeyManager 使用 `os.UserConfigDir()` 获取系统用户配置目录，并在其中创建 `KeyManager` 目录。

典型路径示例：

- Windows：`C:\Users\<User>\AppData\Roaming\KeyManager\`
- macOS：`/Users/<User>/Library/Application Support/KeyManager/`
- Linux：通常位于用户配置目录下的 `KeyManager/`

核心数据文件：

```text
keymanager.sqlite   # SQLite 数据库
secret.key          # API Key 加密密钥
```

`keymanager.sqlite` 和 `secret.key` 必须成对保存。缺少或替换其中任意一个，都可能导致历史 API Key 无法解密。

## 备份与还原

应用内提供“数据备份”功能，会导出一个 zip 文件，包含：

```text
keymanager.sqlite
secret.key
manifest.json
```

还原时会校验 zip 内容、文件名、文件大小、`secret.key` 格式和 SQLite 完整性，然后覆盖当前本地数据。

请注意：

- 备份 zip 包含数据库和加密密钥，拥有该文件的人可能恢复并解密你的 API Key。
- 请将备份文件保存到可信位置，不要上传到公开仓库、公开网盘或聊天记录中。
- 还原是覆盖式操作，不会合并当前数据；还原前建议先备份现有数据。

## 批量导入格式

批量导入支持直接粘贴多行 API Key，也支持逗号分隔格式。

示例：

```text
sk-xxxx
sk-yyyy
```

或：

```text
name,provider,baseUrl,apiKey
OpenAI Prod,openai,https://api.openai.com/v1,sk-xxxx
```

## 批量导出说明

批量导出会生成 TXT 文件，内容包含明文 Base URL 和 API Key。导出的文件应按敏感文件处理，不建议提交到 GitHub 或分享给无关人员。

## 安全说明

- API Key 明文不会直接存入数据库。
- 数据库中保存的是加密后的密文、nonce 和脱敏展示值。
- `secret.key` 是解密 API Key 的关键文件，应与数据库一起保护。
- 备份 zip 与批量导出的 TXT 都属于敏感文件。
- 审计日志和界面提示不会记录完整 API Key 明文。

## 发布到 GitHub 前建议

发布前建议检查以下事项：

```bash
npm --prefix frontend run build
go test ./...
wails build
```

同时确认不要提交以下敏感或临时内容：

- 本机 `keymanager.sqlite`
- 本机 `secret.key`
- 备份 zip
- 批量导出的 TXT
- 临时构建脚本或测试数据
- `frontend/node_modules/`

## License

当前项目尚未声明许可证。发布到 GitHub 前，建议根据你的分发需求补充 `LICENSE` 文件。
