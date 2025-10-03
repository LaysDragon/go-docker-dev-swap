# docker-dev-swap

一個用於微服務開發的容器替換調試工具，支持遠端 Docker Compose 環境的快速開發和調試。

## 使用情景
接手大型遠古微服務專案，重建完整的遠端編譯環境不夠方便，私有庫認證、依賴同步等問題層出不窮。直接端口轉發也不適用，因為這些微服務通過共享掛載目錄緊密耦合。

本工具採用另一種方案：本地快速編譯後，將執行檔和調試器上傳至測試伺服器，動態替換目標容器並啟動 Delve debugger，透過 SSH tunnel 實現遠端調試。退出時自動恢復原始環境。

**跨平台編譯注意事項：**
- 不同環境編譯可能存在動態庫依賴問題，建議使用相同環境編譯或在 `initial_scripts` 中安裝必要依賴
- Delve 需要自行提供，可複製本地安裝版本或指定路徑。目標容器如內建 dlv 更佳，但仍需注意依賴問題
- 範例：本地 WSL Ubuntu + 目標 Alpine 容器，只需通過 `initial_scripts` 配置在容器中執行 `apk add --no-cache libc6-compat` 即可解決依賴

## 功能特性

- 🔄 **容器替換**: 自動停止原始容器，建立配置相同的開發容器
- 🐛 **遠端調試**: 提供 SSH tunnel 連接在本地上暴露 Delve debugger 端口
- 📦 **自動部署**: 監控本地檔案，自動上傳並重啟容器
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


### 自定義 Delve 參數
更多配置請參考 [CONFIG.md](docs/CONFIG.md)
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

## 授權

MIT License

## 貢獻

歡迎提交 Issue 和 Pull Request!