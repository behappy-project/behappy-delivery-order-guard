# 使用示例

## 启动 Redis（如果还没有运行）

### 使用 Docker
```bash
docker run -d -p 6379:6379 --name redis redis:latest
```

### 使用本地 Redis
```bash
redis-server
```

## 运行系统

```bash
# 方式1: 直接运行
go run main.go

# 方式2: 使用 Makefile
make run

# 方式3: 先构建再运行
make build
./bin/delivery-order-guard
```

## 查看日志输出

系统运行时会输出以下类型的日志：

1. **订单接收日志**
```
{"level":"info","timestamp":"2024-01-01T10:00:00Z","msg":"Received Meituan order","order_id":"MT1234567890","order_no":"MT12345678","amount":52}
```

2. **打印处理日志**
```
{"level":"info","timestamp":"2024-01-01T10:00:05Z","msg":"Order printed successfully","worker_id":0,"order_id":"MT1234567890","duration":"500ms"}
```

3. **超时报警日志**
```
{"level":"warn","timestamp":"2024-01-01T10:15:30Z","msg":"Order timeout detected","order_id":"MT1234567890","elapsed":"15m30s"}
{"level":"error","timestamp":"2024-01-01T10:15:30Z","msg":"ALERT: Order timeout","order_id":"MT1234567890","message":"⚠️ 订单超时报警\n..."}
```

## 检查 Redis 中的订单数据

```bash
# 连接到 Redis
redis-cli

# 查看所有订单键
KEYS order:*

# 查看特定订单
GET order:MT1234567890

# 查看订单状态
GET order:status:MT1234567890
```

## 自定义配置

编辑 `config.yaml` 文件来调整：

- 订单接收频率（`platforms[].interval`）
- 打印工作协程数（`printer_workers`）
- 订单超时时间（`order_timeout`）
- Redis 连接信息

修改后重启系统即可生效。

