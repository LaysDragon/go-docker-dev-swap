# Executor 包结构说明

## 概述

`executor` 包提供了统一的命令执行、文件操作和 SSH tunnel 管理接口，支持本地和远程两种执行模式。

## 文件结构

```
internal/executor/
├── interface.go          # 接口定义（Session、Executor、TunnelCloser）
├── factory.go            # 工厂方法（根据配置创建对应的 Executor）
├── local.go              # 本地执行器实现
├── local_session.go      # 本地 Session 实现
├── remote.go             # 远程执行器实现
├── remote_session.go     # 远程 Session 实现
└── util.go               # 工具类型（noopCloser）
```

## 各文件说明

### interface.go (49 行)
定义核心接口：

- **Session**: 流式命令执行接口
  - `StdoutPipe()` - 获取标准输出
  - `Start()` - 启动命令
  - `Wait()` - 等待完成
  - `Close()` - 关闭 session

- **Executor**: 执行器接口
  - `Execute()` - 执行 shell 命令
  - `CreateSession()` - 创建流式 session
  - `UploadFile()` - 上传/复制文件
  - `CreateScript()` - 创建脚本
  - `CreateTunnel()` - 创建 SSH tunnel
  - `Close()` - 关闭连接
  - `IsRemote()` - 判断模式

- **TunnelCloser**: Tunnel 关闭接口

## 设计模式

### 1. 接口隔离原则 (ISP)
- 清晰的接口定义分离在 `interface.go`
- 实现细节分散在各自的文件中

### 2. 依赖倒置原则 (DIP)
- 上层依赖 `Executor` 接口，不依赖具体实现
- `factory.go` 负责创建具体实现

### 3. 适配器模式
- `LocalSession` 和 `RemoteSession` 适配不同的底层 API
- 统一的 `Session` 接口隐藏差异

### 4. 工厂模式
- `NewExecutor()` 根据配置创建对应实例
- `NewLocalExecutor()` 和 `NewRemoteExecutor()` 分别构造

## 使用示例

```go
import "github.com/laysdragon/go-docker-dev-swap/internal/executor"

// 1. 创建 executor
exec, err := executor.NewExecutor(cfg)
if err != nil {
    return err
}
defer exec.Close()

// 2. 执行命令
output, err := exec.Execute("docker ps")

// 3. 流式命令执行
session, err := exec.CreateSession()
defer session.Close()

session.Start("docker logs -f container")
stdout, _ := session.StdoutPipe()

scanner := bufio.NewScanner(stdout)
for scanner.Scan() {
    fmt.Println(scanner.Text())
}

session.Wait()

// 4. 文件操作
err = exec.UploadFile("/local/path", "/remote/path")

// 5. 创建 tunnel（仅远程模式）
if exec.IsRemote() {
    tunnel, err := exec.CreateTunnel(2345, 2345)
    defer tunnel.Close()
}
```

## 代码统计

| 文件 | 行数 | 功能 |
|-----|------|------|
| interface.go | 49 | 接口定义 |
| factory.go | 13 | 工厂方法 |
| local.go | 105 | 本地执行器 |
| local_session.go | 50 | 本地 Session |
| remote.go | 64 | 远程执行器 |
| remote_session.go | 51 | 远程 Session |
| util.go | 8 | 工具类型 |
| **总计** | **340** | - |

## 优势

### ✅ 模块化清晰
- 每个文件职责单一
- 易于定位和修改

### ✅ 可维护性高
- 本地和远程实现完全分离
- Session 实现独立管理

### ✅ 易于测试
- 可以针对每个文件单独测试
- Mock 接口简单

### ✅ 易于扩展
- 添加新功能只需修改对应文件
- 接口变更影响范围明确

## 未来扩展

如果需要添加新的执行模式（如 Docker exec、Kubernetes exec）：

1. 在 `interface.go` 中保持接口不变
2. 创建新的实现文件：
   - `docker.go` - Docker executor 实现
   - `docker_session.go` - Docker session 实现
3. 在 `factory.go` 中添加创建逻辑
4. 上层代码无需任何修改

## 相关文档

- [SESSION_ABSTRACTION.md](../../docs/SESSION_ABSTRACTION.md) - Session 抽象设计
- [SESSION_API_DESIGN.md](../../docs/SESSION_API_DESIGN.md) - API 调用顺序设计
- [MODES.md](../../docs/MODES.md) - 执行模式说明
