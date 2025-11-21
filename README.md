# docker-dev-swap

一個用於微服務開發的容器替換調試工具，支持遠端 Docker 容器與 Compose 環境的快速開發和調試。  
將本地編譯執行檔和調試器上傳至測試伺服器，動態替換目標容器並啟動 Delve debugger，透過 SSH tunnel 實現遠端調試。退出時自動恢復原始環境。
僅支持 linux，windows 上請使用 wsl。

**跨平台編譯注意事項：**
- 不同環境編譯可能存在動態庫依賴問題，建議使用相同環境編譯或在 `initial_scripts` 中安裝必要依賴。
- Delve 需自行提供，可複製本地安裝版本或指定路徑，需注意依賴問題。目標容器如內建 dlv 更佳。
 
> 範例：本地 WSL Ubuntu + 目標 Alpine 容器。  
> 只需在 `initial_scripts` 配置 `apk add --no-cache libc6-compat` 即可處理缺少動態庫的錯誤。

## 功能特性

- 🔄 **容器替換**: 自動停止原始容器，建立配置相同的開發容器
- 🏠 **雙模式支持**: 支持本地和遠端兩種執行模式，靈活切換
- 🐛 **遠端調試**: 提供 SSH tunnel 連接在本地上暴露 Delve debugger 端口
- 📦 **自動部署**: 監控本地檔案，自動上傳並重啟容器
- 🧹 **自動清理**: 退出時自動清理開發容器並恢復原始服務
- 📝 **日誌監控**: 實時監控容器日誌，可選寫入本地文件

## 安裝

```bash
go install github.com/laysdragon/go-docker-dev-swap@latest
```

或從源碼構建:

```bash
git clone https://github.com/laysdragon/go-docker-dev-swap.git
cd go-docker-dev-swap
go build -o go-docker-dev-swap ./cmd/go-docker-dev-swap
```

## 使用方法

### 1. 建立配置檔案

複製 `config.example.yaml` 為 `config.yaml` 並依需求新增 **components / hosts / projects**：

```yaml
log_file: ""
dlv_config:
  enabled: true
  port: 2345

components:
  api-service:
    name: "API"
    local_binary: "./bin/api"
    target_service: "api"
    container_binary_path: "/app/api"

hosts:
  dev-server:
    name: "DEV"
    mode: "remote"
    host: "your-server.com"
    user: "developer"
    password: "your-password"      # 或 key_file
    remote_work_dir: "/tmp/dev"
    projects:
      main-compose:
        name: "Compose"
        type: "compose"
        compose_dir: "/path/to/docker-compose"

  local-docker:
    name: "Local"
    mode: "local"                  # 本地執行不需要 SSH 欄位
    projects:
      playground:
        type: "compose"
        compose_dir: "./deploy"
```

- Component 配置「要替換的目標服務容器與本地編譯的組件執行檔」。
- Host 描述執行環境（`mode` 可為 `remote` 或 `local`）以及在該環境可用的專案選項。
- Project 可為 `type=compose` 或 `type=container`。
- 啟動程式後，會依序互動式選擇 component → host → project；若某步只有單一選項會自動略過。

> 完整欄位說明請參考 [CONFIG.md](docs/CONFIG.md)，模式細節請參考 [MODES.md](docs/MODES.md)。

### 2. 編譯你的 Go 應用

確保編譯時包含調試資訊:

```bash
go build -gcflags="all=-N -l" -o ./bin/your-app ./cmd/your-app
```

### 3. 啟動開發環境

```bash
go-docker-dev-swap
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
3. 工具自動偵測並上傳新執行檔，並重啟容器

### 6. 退出

按 `Ctrl+C` 退出，工具會自動清理暫時性容器並恢復原始容器服務:

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

在對應的 host 區塊中提供 `key_file` 即可：

```yaml
hosts:
  dev-server:
    mode: "remote"
    host: "your-server.com"
    user: "developer"
    key_file: "/home/user/.ssh/id_rsa"
    projects:
      main:
        compose_dir: "/path/to/docker-compose"
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