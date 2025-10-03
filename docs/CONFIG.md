# 配置指南

## 配置系統
## 配置文件位置

程序會按以下順序查找配置文件：

1. 命令行指定的路徑：`--config=/path/to/config.yaml`
2. 當前目錄：`./config.yaml`
3. 用戶主目錄：`~/.docker-dev-swap/config.yaml`
4. 系統目錄：`/etc/docker-dev-swap/config.yaml`

## 配置項說明
### 必要配置項

這些配置項必須設定，否則程序無法運行：

| 配置項 | 說明 | 範例 |
|--------|------|------|
| `remote_host.host` | 遠端主機地址 | `"192.168.1.100"` |
| `remote_host.user` | SSH 用戶名 | `"developer"` |
| `remote_host.password` 或 `remote_host.key_file` | SSH 認證（二選一） | `"password"` 或 `"~/.ssh/id_rsa"` |
| `compose_dir` | docker-compose.yml 所在目錄 | `"/opt/app"` |
| `target_service` | 目標服務名稱 | `"api-service"` |
| `local_binary` | 本地二進制文件路徑 | `"./bin/app"` |

### 可選配置項（有預設值）

| 配置項 | 說明 | 預設值 |
|--------|------|--------|
| `remote_host.port` | SSH 端口 | `22` |
| `remote_work_dir` | 遠端工作目錄 | `"/tmp/dev-binaries"` |
| `remote_binary_name` | 遠端執行檔名稱 | `"service"` |
| `container_binary_path` | 容器內執行檔路徑 | `"/app/service"` |
| `debugger_port` | 本地 debugger 端口 | `2345` |
| `extra_ports` | 額外暴露的端口 | `[]` |
| `initial_scripts` | 容器初始化腳本 | `""` |
| `dlv_config.enabled` | 是否啟用 dlv | `true` |
| `dlv_config.port` | dlv 端口 | `2345` |
| `dlv_config.local_path` | 本地 dlv 路徑（空則自動搜尋） | `""` |
| `dlv_config.args` | dlv 額外參數 | `""` |
| `log_file` | 本地日誌文件路徑 | `""` (不寫入文件) |

## 環境變數支持

所有配置項都可以通過環境變數覆蓋，使用前綴 `DDS_`（Docker Dev Swap）：

```bash
# 設定遠端主機
export DDS_REMOTE_HOST_HOST="192.168.1.100"
export DDS_REMOTE_HOST_USER="developer"
export DDS_REMOTE_HOST_PASSWORD="secret"

# 設定目標服務
export DDS_TARGET_SERVICE="api-service"
export DDS_COMPOSE_DIR="/opt/app"

# 設定本地二進制文件
export DDS_LOCAL_BINARY="./bin/app"

# 設定 debugger 端口
export DDS_DEBUGGER_PORT="3000"

# 啟動程序
./docker-dev-swap
```

環境變數命名規則：
- 使用下劃線分隔
- 巢狀配置使用雙下劃線
- 全部大寫

範例：
- `remote_host.port` → `DDS_REMOTE_HOST_PORT`
- `dlv_config.enabled` → `DDS_DLV_CONFIG_ENABLED`
- `extra_ports` → `DDS_EXTRA_PORTS`

## 最小配置範例

只設定必要項的最小配置：

```yaml
remote_host:
  host: "192.168.1.100"
  user: "developer"
  password: "your-password"

compose_dir: "/opt/app"
target_service: "api-service"
local_binary: "./bin/app"
```

其他配置項都會使用預設值。

## 完整配置範例

包含所有可選項的完整配置：

```yaml
remote_host:
  host: "192.168.1.100"
  port: 22
  user: "developer"
  key_file: "~/.ssh/id_rsa"

compose_dir: "/opt/microservices/my-app"
target_service: "api-service"
local_binary: "./bin/api-service"

remote_work_dir: "/tmp/dev-binaries"
remote_binary_name: "api-service"
container_binary_path: "/app/api-service"

debugger_port: 2345
extra_ports:
  - 8080
  - 9090

initial_scripts: |
  apk add --no-cache libc6-compat
  echo "Container initialized"

dlv_config:
  enabled: true
  port: 2345
  args: "--log --log-output=debugger,rpc"
```

## 配置優先級

配置值的優先級（從高到低）：

1. 環境變數
2. 配置文件
3. 預設值