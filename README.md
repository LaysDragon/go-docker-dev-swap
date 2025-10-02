# docker-dev-swap

一個用於微服務開發的容器替換調試工具，支持遠端 Docker Compose 環境的快速開發和調試。

## 功能特性

- 🔄 **容器替換**: 自動停止原始容器，建立配置相同的開發容器
- 🐛 **遠端調試**: 內建 Delve debugger 支持，通過 SSH tunnel 連接
- 📦 **自動部署**: 監控本地編譯檔案，自動上傳並重啟容器
- 🧹 **自動清理**: 退出時自動清理開發容器並恢復原始服務
- 🔌 **SSH 管理**: 內建 SSH 和 SFTP 支持，無需額外工具

## 安裝

```bash
go install github.com/laysdragon/go-docker-dev-swap@latest
```

或從源碼構建:

```bash
git clone https://github.com/laysdragon/go-docker-dev-swap.git
cd docker-dev-swap
go build -o docker-dev-swap ./cmd/docker-dev-swap
```

## 使用方法

### 1. 建立配置檔案

複製 `config.example.yaml` 為 `config.yaml` 並修改:

```yaml
remote_host:
  host: "your-server.com"
  user: "developer"
  password: "your-password"

compose_dir: "/path/to/docker-compose"
target_service: "your-service"
local_binary: "./bin/your-app"
container_binary_path: "/app/your-app"
```

### 2. 編譯你的 Go 應用

確保編譯時包含調試資訊:

```bash
go build -gcflags="all=-N -l" -o ./bin/your-app ./cmd/your-app
```

### 3. 啟動開發環境

```bash
docker-dev-swap -config config.yaml
```

或指定特定服務:

```bash
docker-dev-swap -config config.yaml -service api-service
```

### 4. 連接 Debugger

在 IDE 中配置遠端調試:

**GoLand / IntelliJ IDEA:**
- Run → Edit Configurations → Add New → Go Remote
- Host: `localhost`
- Port: `2345`

**VS Code (.vscode/launch.json):**
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Connect to Remote Delve",
      "type": "go",
      "request": "attach",
      "mode": "remote",
      "remotePath": "/app",
      "port": 2345,
      "host": "localhost"
    }
  ]
}
```

### 5. 開發工作流

1. 修改代碼
2. 在本地編譯: `go build -gcflags="all=-N -l" -o ./bin/your-app`
3. 工具自動偵測並上傳新執行檔
4. 容器自動重啟
5. Debugger 重新連接

### 6. 退出

按 `Ctrl+C` 退出，工具會自動:
- 停止並刪除開發容器
- 恢復原始容器
- 關閉 SSH tunnel

## 進階配置

### 多端口暴露

```yaml
extra_ports:
  - 8080  # HTTP
  - 9090  # Metrics
  - 6060  # pprof
```

### 自定義 Delve 參數

```yaml
dlv_config:
  enabled: true
  port: 2345
  args: "--log --log-output=debugger,rpc,dap"
```

### 使用 SSH 金鑰認證

```yaml
remote_host:
  host: "your-server.com"
  user: "developer"
  key_file: "/home/user/.ssh/id_rsa"
  # 註解掉 password
```

## 工作原理

```
┌─────────────┐      ┌──────────────────┐      ┌─────────────┐
│  本地開發   │ SSH  │   遠端主機        │      │  Docker     │
│             │─────>│                  │      │             │
│ 1. 編譯 Go  │      │ 2. 停止原始容器   │<─────│ Original    │
│ 2. 自動偵測 │      │ 3. 建立開發容器   │      │ Container   │
│ 3. 上傳檔案 │ SCP  │ 4. 掛載執行檔     │      │             │
│             │─────>│ 5. 啟動 Delve    │      │ Dev         │
│ 4. 連接調試 │Tunnel│ 6. SSH Tunnel    │<─────│ Container   │
│             │<─────│                  │      │ (with dlv)  │
└─────────────┘      └──────────────────┘      └─────────────┘
```

## 故障排除

### SSH 連接失敗
- 檢查主機、端口、用戶名和密碼
- 確認防火牆規則
- 嘗試使用 `ssh user@host` 手動測試

### 容器啟動失敗
- 檢查原始容器是否正常運行
- 確認 `compose_dir` 路徑正確
- 查看遠端 Docker 日誌: `docker logs <container-name>-dev`

### Debugger 無法連接
- 確認 SSH tunnel 已建立
- 檢查防火牆是否阻擋本地端口
- 確認容器內 Delve 正常運行: `docker exec <container-name>-dev ps aux | grep dlv`

### 檔案上傳失敗
- 檢查 `remote_binary_path` 目錄是否有寫入權限
- 確認磁碟空間充足

## 依賴項目

- [golang.org/x/crypto/ssh](https://pkg.go.dev/golang.org/x/crypto/ssh) - SSH 客戶端
- [github.com/pkg/sftp](https://github.com/pkg/sftp) - SFTP 檔案傳輸
- [github.com/fsnotify/fsnotify](https://github.com/fsnotify/fsnotify) - 檔案監控
- [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) - YAML 解析

## 授權

MIT License

## 貢獻

歡迎提交 Issue 和 Pull Request!