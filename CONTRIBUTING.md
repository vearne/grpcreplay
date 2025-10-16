# 贡献指南 (Contributing Guide)

感谢你对 grpcreplay 项目的关注！我们欢迎任何形式的贡献。

## 🚀 快速开始

### 环境要求

- Go 1.23+ 
- libpcap 开发库
- Git

### 安装依赖

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install -y libpcap-dev
```

**CentOS/RHEL:**
```bash
sudo yum install -y libpcap-devel
```

**macOS:**
```bash
brew install libpcap
```

### 克隆和构建

```bash
git clone https://github.com/vearne/grpcreplay.git
cd grpcreplay
go mod download
make build
```

## 📋 贡献类型

我们欢迎以下类型的贡献：

- 🐛 **Bug 修复**
- ✨ **新功能**
- 📚 **文档改进**
- 🧪 **测试用例**
- 🎨 **代码优化**
- 🔧 **工具和脚本**

## 🔄 贡献流程

### 1. Fork 项目

点击项目页面右上角的 "Fork" 按钮

### 2. 创建功能分支

```bash
git checkout -b feature/your-feature-name
# 或者修复bug
git checkout -b fix/your-bug-fix
```

### 3. 进行开发

- 遵循现有的代码风格
- 添加必要的测试
- 更新相关文档
- 确保所有测试通过

### 4. 提交代码

```bash
git add .
git commit -m "feat: add your feature description"
# 或者
git commit -m "fix: fix your bug description"
```

**提交信息格式：**
- `feat:` 新功能
- `fix:` Bug修复
- `docs:` 文档更新
- `test:` 测试相关
- `refactor:` 代码重构
- `perf:` 性能优化
- `chore:` 构建过程或辅助工具的变动

### 5. 推送到你的 Fork

```bash
git push origin feature/your-feature-name
```

### 6. 创建 Pull Request

- 在 GitHub 上创建 Pull Request
- 填写详细的描述
- 关联相关的 Issue（如果有）

## 🧪 测试

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行测试并生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 添加测试

- 为新功能添加单元测试
- 确保测试覆盖率不降低
- 测试文件命名为 `*_test.go`

### 测试示例

```go
func TestYourFunction(t *testing.T) {
    // 准备测试数据
    input := "test input"
    expected := "expected output"
    
    // 执行测试
    result := YourFunction(input)
    
    // 验证结果
    if result != expected {
        t.Errorf("Expected %s, got %s", expected, result)
    }
}
```

## 📝 代码规范

### Go 代码风格

- 使用 `gofmt` 格式化代码
- 使用 `golangci-lint` 进行代码检查
- 遵循 [Effective Go](https://golang.org/doc/effective_go.html) 指南

### 命名规范

- 包名：小写，简短，有意义
- 函数名：驼峰命名，公开函数首字母大写
- 变量名：驼峰命名，简洁明了
- 常量：全大写，下划线分隔

### 注释规范

- 公开的函数、类型、常量必须有注释
- 注释以函数名开头
- 复杂逻辑需要添加行内注释

```go
// ProcessPacket 处理网络数据包并返回解析后的结果
// 参数 packet 是原始网络数据包
// 返回值包含解析后的数据和可能的错误
func ProcessPacket(packet []byte) (*ParsedData, error) {
    // 实现逻辑...
}
```

## 🐛 Bug 报告

在提交 Bug 报告时，请包含：

- **环境信息**：操作系统、Go版本、grpcreplay版本
- **重现步骤**：详细的操作步骤
- **期望行为**：你期望发生什么
- **实际行为**：实际发生了什么
- **错误日志**：相关的错误信息
- **配置文件**：如果相关的话

### Bug 报告模板

```markdown
## 环境信息
- OS: [e.g. Ubuntu 20.04]
- Go Version: [e.g. 1.23.0]
- grpcreplay Version: [e.g. v0.2.10]

## 重现步骤
1. 执行命令 `./grpcr --input-raw="127.0.0.1:8080" --output-stdout`
2. 发送 gRPC 请求
3. 观察输出

## 期望行为
应该正确捕获并显示 gRPC 请求

## 实际行为
程序崩溃并显示错误信息

## 错误日志
```
panic: runtime error: invalid memory address
```

## 配置
使用默认配置
```

## 💡 功能请求

在提交功能请求时，请：

- 描述你想要的功能
- 解释为什么需要这个功能
- 提供使用场景
- 如果可能，提供设计建议

## 📖 文档贡献

文档改进包括：

- 修复错别字和语法错误
- 改进现有文档的清晰度
- 添加新的使用示例
- 翻译文档

## 🔍 代码审查

所有的 Pull Request 都需要经过代码审查：

- 至少一个维护者的批准
- 所有 CI 检查通过
- 解决所有审查意见

## 📞 联系方式

如果你有任何问题，可以通过以下方式联系我们：

- 创建 GitHub Issue
- 发送邮件给维护者
- 在相关的 Pull Request 中评论

## 🙏 致谢

感谢所有为 grpcreplay 项目做出贡献的开发者！

---

**注意：** 通过提交代码，你同意你的贡献将在项目的开源许可证下发布。
