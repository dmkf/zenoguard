# 智巡Guard API文档

## 基础信息

- **Base URL**: `https://monitor.example.com/api`
- **认证方式**: Bearer Token
- **数据格式**: JSON

---

## Agent接口

### 上报数据

**Endpoint**: `POST /agent/report`

**认证**: Bearer Token (主机Token)

**请求头**:
```
Content-Type: application/json
Authorization: Bearer {token}
```

**请求体**:
```json
{
  "hostname": "server-01",
  "ssh_logins": [
    {
      "user": "root",
      "ip": "1.2.3.4",
      "time": "2026-01-30 10:00:00",
      "method": "password",
      "success": true,
      "port": 22,
      "protocol": "ssh2"
    }
  ],
  "system_load": {
    "load1": 0.5,
    "load5": 0.8,
    "load15": 0.6
  },
  "network_traffic": {
    "interface": "eth0",
    "in_bytes": 1234567,
    "out_bytes": 7654321,
    "total_bytes": 8888888,
    "bandwidth_mbps": 12.5
  },
  "public_ip": "1.2.3.4"
}
```

**响应**:
```json
{
  "success": true,
  "report_interval": 60
}
```

**错误响应**:
- `401 Unauthorized`: Token无效
- `403 Forbidden`: 主机已停用
- `400 Bad Request`: 请求参数错误

---

## 管理接口

### 认证接口

#### 登录
**Endpoint**: `POST /auth/login`

**请求体**:
```json
{
  "username": "admin",
  "password": "admin123"
}
```

**响应**:
```json
{
  "user": {
    "id": 1,
    "username": "admin"
  },
  "token": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}
```

#### 登出
**Endpoint**: `POST /auth/logout`

**认证**: Bearer Token

**响应**:
```json
{
  "message": "Logged out successfully"
}
```

#### 获取当前用户
**Endpoint**: `GET /auth/me`

**认证**: Bearer Token

**响应**:
```json
{
  "user": {
    "id": 1,
    "username": "admin"
  }
}
```

---

### 主机管理接口

#### 主机列表
**Endpoint**: `GET /hosts`

**认证**: Bearer Token

**查询参数**:
- `is_active`: boolean (可选)
- `search`: string (可选)
- `page`: integer (默认1)
- `per_page`: integer (默认20)

**响应**:
```json
{
  "data": [
    {
      "id": 1,
      "hostname": "server-01",
      "token": "abc123...",
      "remark": "Web服务器",
      "report_interval": 60,
      "alert_rules": "SSH登录失败超过5次",
      "is_active": true,
      "created_at": "2026-01-30T10:00:00.000000Z",
      "updated_at": "2026-01-30T10:00:00.000000Z",
      "latest_data": {...}
    }
  ],
  "current_page": 1,
  "total": 10
}
```

#### 创建主机
**Endpoint**: `POST /hosts`

**认证**: Bearer Token

**请求体**:
```json
{
  "hostname": "server-02",
  "remark": "数据库服务器",
  "report_interval": 120,
  "alert_rules": "系统负载超过5.0",
  "is_active": true
}
```

**响应**: 返回创建的主机对象（包含生成的token）

#### 获取主机详情
**Endpoint**: `GET /hosts/{id}`

**认证**: Bearer Token

#### 更新主机
**Endpoint**: `PUT /hosts/{id}`

**认证**: Bearer Token

**请求体**: 同创建主机（所有字段可选）

#### 删除主机
**Endpoint**: `DELETE /hosts/{id}`

**认证**: Bearer Token

#### 获取主机数据
**Endpoint**: `GET /hosts/{id}/data`

**认证**: Bearer Token

**查询参数**:
- `start_date`: string (ISO 8601格式)
- `end_date`: string (ISO 8601格式)
- `is_alert`: boolean
- `search`: string
- `page`: integer
- `per_page`: integer

**响应**:
```json
{
  "data": [
    {
      "id": 1,
      "host_id": 1,
      "report_time": "2026-01-30T10:00:00.000000Z",
      "ssh_logins": [...],
      "system_load": {...},
      "network_traffic": {...},
      "public_ip": "1.2.3.4",
      "ip_location": "中国浙江杭州",
      "llm_summary": "服务器运行正常",
      "is_alert": false,
      "created_at": "2026-01-30T10:00:00.000000Z"
    }
  ],
  "current_page": 1,
  "total": 100
}
```

#### 重新生成Token
**Endpoint**: `POST /hosts/{id}/regenerate-token`

**认证**: Bearer Token

**响应**:
```json
{
  "token": "new_token_here"
}
```

---

### LLM配置接口

#### 获取LLM配置
**Endpoint**: `GET /llm`

**认证**: Bearer Token

**响应**:
```json
{
  "data": {
    "id": 1,
    "model_name": "gpt-4",
    "api_url": "https://api.openai.com/v1/chat/completions",
    "is_active": true
  }
}
```

#### 更新LLM配置
**Endpoint**: `PUT /llm`

**认证**: Bearer Token

**请求体**:
```json
{
  "model_name": "gpt-3.5-turbo",
  "api_url": "https://api.openai.com/v1/chat/completions",
  "api_key": "sk-...",
  "is_active": true
}
```

#### 测试LLM连接
**Endpoint**: `POST /llm/test`

**认证**: Bearer Token

**响应**:
```json
{
  "success": true,
  "message": "Connection successful"
}
```

---

### 预警配置接口

#### 获取预警配置列表
**Endpoint**: `GET /alert`

**认证**: Bearer Token

**响应**:
```json
{
  "data": [
    {
      "id": 1,
      "platform": "dingtalk",
      "webhook_url": "https://oapi.dingtalk.com/robot/send?access_token=...",
      "is_active": true
    }
  ]
}
```

#### 更新预警配置
**Endpoint**: `PUT /alert`

**认证**: Bearer Token

**请求体**:
```json
{
  "platform": "dingtalk",
  "webhook_url": "https://oapi.dingtalk.com/robot/send?access_token=...",
  "secret": "SEC...",
  "is_active": true
}
```

#### 测试预警
**Endpoint**: `POST /alert/test`

**认证**: Bearer Token

**请求体**:
```json
{
  "platform": "dingtalk"
}
```

#### 删除预警配置
**Endpoint**: `DELETE /alert/{id}`

**认证**: Bearer Token

---

## 数据模型

### Host (主机)
| 字段 | 类型 | 说明 |
|------|------|------|
| id | integer | 主机ID |
| hostname | string | 主机名 |
| token | string | 认证Token (64位) |
| remark | string | 备注 |
| report_interval | integer | 上报间隔(秒) |
| alert_rules | string | 预警规则 |
| is_active | boolean | 是否活跃 |
| created_at | timestamp | 创建时间 |
| updated_at | timestamp | 更新时间 |

### HostData (主机数据)
| 字段 | 类型 | 说明 |
|------|------|------|
| id | integer | 数据ID |
| host_id | integer | 主机ID |
| report_time | timestamp | 上报时间 |
| ssh_logins | json | SSH登录记录 |
| system_load | json | 系统负载 |
| network_traffic | json | 网络流量 |
| public_ip | string | 公网IP |
| ip_location | string | IP地理位置 |
| llm_summary | string | LLM分析摘要 |
| is_alert | boolean | 是否预警 |
| created_at | timestamp | 创建时间 |

### SSHLogin (SSH登录)
| 字段 | 类型 | 说明 |
|------|------|------|
| user | string | 用户名 |
| ip | string | IP地址 |
| time | string | 时间 |
| method | string | 认证方式 |
| success | boolean | 是否成功 |
| port | integer | 端口 |
| protocol | string | 协议 |

### SystemLoad (系统负载)
| 字段 | 类型 | 说明 |
|------|------|------|
| load1 | float | 1分钟平均负载 |
| load5 | float | 5分钟平均负载 |
| load15 | float | 15分钟平均负载 |

### NetworkTraffic (网络流量)
| 字段 | 类型 | 说明 |
|------|------|------|
| interface | string | 网卡名称 |
| in_bytes | integer | 入站字节数 |
| out_bytes | integer | 出站字节数 |
| total_bytes | integer | 总字节数 |
| bandwidth_mbps | float | 带宽(Mbps) |

---

## 错误码

| 错误码 | 说明 |
|--------|------|
| 400 | 请求参数错误 |
| 401 | 未认证或Token无效 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 422 | 验证失败 |
| 500 | 服务器错误 |

错误响应格式：
```json
{
  "error": "错误信息",
  "errors": {
    "field": ["验证错误详情"]
  }
}
```
