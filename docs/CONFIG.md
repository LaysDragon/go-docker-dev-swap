# 配置指南

`docker-dev-swap` 支援「多組配置」格式。  
每個配置檔可同時定義多個 **components**（本地組件）、**hosts**（執行節點）以及各 host 底下的 **projects**。    
啟動程式時，會依序選擇 component → host → project，並組成執行所需的 runtime config。

## 配置檔案搜尋順序

1. 指定參數：`--config=/path/to/docker-dev-swap.yaml`
2. 當前目錄：`./docker-dev-swap.yaml`
3. 使用者目錄：`~/.docker-dev-swap.yaml`

## 結構概覽

```yaml
log_file: ""             # 全域預設值（可被 component 覆寫）
initial_scripts: ""       # 全域預設值（可被 component 覆寫）
dlv_config: # 全域預設值（可被 component 覆寫）
  enabled: false
  port: 2345
  args: ""
  local_path: ""

components: { ... }      # 至少一個 component
hosts: { ... }           # 至少一個 host，每個 host 需含 projects
```

- **Component**：描述要熱替換的二進制，以及容器內對應的 service。
- **Host**：描述要在哪裡執行（本機或遠端）、SSH / sudo / docker。
    - **Project** 該 host 上可用的 docker-compose projects。

## Component 欄位
通常為本地開發環境的golang程序，以及需要替換的目標容器

| 欄位                      | 說明                                                  | 必要 | 預設值            |
|-------------------------|-----------------------------------------------------|----|----------------|
| `name`                  | 顯示名稱（互動選單用）                                         | 否  | key 名稱         |
| `local_binary`          | 本地編譯好的二進制路徑                                         | ✅  | —              |
| `target_service`        | docker-compose service 名稱                           | ✅  | —              |
| `container_binary_path` | 容器內二進制儲存路徑                                          | 否  | `/app/service` |
| `debugger_port`         | 用於為 Delve/debugger 暴露的本地埠建立通道，通常情況下應與 dlv_config 一致 | 否  | `2345`         |
| `dlv_config`            | 覆蓋全域 dlv 設定                                         | 否  | 全域設定           |
| `initial_scripts`       | 容器啟動後執行的腳本                                          | 否  | 全域設定           |
| `log_file`              | 追加輸出的本地檔案路徑                                         | 否  | 全域設定           |

## Host

host 代表實際執行環境，可為 遠端ssh`remote` 或 本地`local`。

### Host 共用欄位

| 欄位                       | 說明                                            | 必要 | 預設值                 |
|--------------------------|-----------------------------------------------|----|---------------------|
| `name`                   | 顯示名稱                                          | 否  | key 名稱              |
| `mode`                   | `remote` 或 `local`                            | 否  | `remote`            |
| `remote_work_dir`        | 臨時檔案 / 程式放置路徑                                 | 否  | `/tmp/dev-binaries` |
| `remote_binary_name`     | 上傳到 host 的檔名                                  | 否  | `service`           |
| `use_sudo`               | 是否以 sudo 執行 docker/system 指令                  | 否  | `false`             |
| `sudo_password`          | 是否需要填入自動 sudo 密碼（建議用環境變數 `DDS_SUDO_PASSWORD`） | 否  | `""`                |
| `docker_command`         | docker 指令                                     | 否  | `docker`            |
| `docker_compose_command` | docker compose 指令                             | 否  | `docker compose`    |

### Host Remote 模式

Remote 模式需要額外的配置

| 欄位                      | 說明                          |
|-------------------------|-----------------------------|
| `mode`                  | 必須為`remote`                 |
| `host`                  | SSH 目標主機，例如 `192.168.1.100` |
| `port`                  | SSH 連接埠，預設 `22`             |
| `user`                  | SSH 使用者                     |
| `password` / `key_file` | 兩者擇一提供；key 會轉成絕對路徑          |

### Projects

每個 host 可配置該 host 上擁有的 project。  
主要是默認的Docker容器環境以及Docker Compose專案目錄。

| 欄位            | 說明                                        | 是否必填               |
|---------------|-------------------------------------------|--------------------|
| `name`        | 顯示名稱                                      | 否（預設為 map key）     |
| `type`        | `compose`（預設）或 `container`                | 否                  |
| `compose_dir` | docker-compose.yml 所在目錄（僅 `compose` 類型需要） | `type=compose` 時 ✅ |

`type: container` 允許你直接指向現有的 Docker 容器，而非 docker compose 下的服務容器。

- 選擇此類型後，程式會將 `target_service` 視為目標容器名稱。
- 每台 host 默認擁有一個 key 為 `docker-container`、名稱為 **Docker Container** 的 `container` 專案。
- 若想自訂名稱或描述，可在該 host 的 `projects` map 補上一個 `docker-container` entry 並覆蓋 `name`。

## 環境變數覆蓋

配置欄位都可以用 `DDS_` 前綴的環境變數覆蓋，例如：

```bash
export DDS_COMPONENTS_API_LOCAL_BINARY="./bin/api"
export DDS_HOSTS_DEV_MODE="remote"
export DDS_HOSTS_DEV_PROJECTS_MAIN_COMPOSE_DIR="/opt/app"
```

- 使用底線分隔層級；map key 會轉成大寫（例：`components.api.local_binary` → `DDS_COMPONENTS_API_LOCAL_BINARY`）。
- 若同時存在環境變數與配置檔，**環境變數優先**。

## 配置範例
請查閱 `docker-dev-swap.example.yaml`

## 配置優先順序

1. 環境變數 (`DDS_`)
2. 配置檔內容
3. 內建預設值（程式碼中的 defaultValues）