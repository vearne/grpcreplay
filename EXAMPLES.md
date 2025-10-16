# grpcreplay 使用示例和最佳实践

## 📚 目录

- [基础使用](#基础使用)
- [高级功能](#高级功能)
- [生产环境部署](#生产环境部署)
- [性能调优](#性能调优)
- [故障排查](#故障排查)
- [最佳实践](#最佳实践)

## 🚀 基础使用

### 1. 简单的流量捕获

**场景**: 捕获本地 gRPC 服务的流量并在控制台显示

```bash
# 捕获 localhost:8080 的 gRPC 流量
sudo ./grpcr --input-raw="127.0.0.1:8080" --output-stdout

# 同时记录请求和响应
sudo ./grpcr --input-raw="127.0.0.1:8080" --output-stdout --record-response
```

**输出示例**:
```
2 f8762dc4-20fa-11f0-a55f-5626e1cdcfe2 1745492273089274000 1
/SearchService/CurrentTime
{"headers":{":authority":"127.0.0.1:8080",":method":"POST",":path":"/SearchService/CurrentTime",":scheme":"http","content-type":"application/grpc"},"body":"{\"requestId\":\"2\"}"}
{"headers":{":status":"200","content-type":"application/grpc","grpc-status":"0"},"body":"{\"currentTime\":\"2025-04-24T18:57:49+08:00\"}"}
```

### 2. 流量转发和复制

**场景**: 将生产环境的流量复制到测试环境

```bash
# 捕获生产环境流量并转发到测试环境
sudo ./grpcr \
  --input-raw="0.0.0.0:8080" \
  --output-grpc="grpc://test-server:8080" \
  --output-stdout
```

### 3. 流量录制

**场景**: 将流量保存到文件以便后续分析

```bash
# 录制到文件，每个文件最大 100MB
sudo ./grpcr \
  --input-raw="127.0.0.1:8080" \
  --output-file-directory="/tmp/grpc-capture" \
  --output-file-max-size=100 \
  --record-response
```

### 4. 流量重放

**场景**: 从录制的文件重放流量

```bash
# 以原始速度重放
./grpcr \
  --input-file-directory="/tmp/grpc-capture" \
  --output-grpc="grpc://target-server:8080" \
  --output-stdout

# 以 10 倍速重放
./grpcr \
  --input-file-directory="/tmp/grpc-capture" \
  --output-grpc="grpc://target-server:8080" \
  --input-file-replay-speed=10
```

## 🔧 高级功能

### 1. 流量过滤

**按方法名过滤**:
```bash
# 只捕获方法名以 "Get" 开头的请求
sudo ./grpcr \
  --input-raw="127.0.0.1:8080" \
  --output-stdout \
  --include-filter-method-match="^.*Get.*$"

# 只捕获特定服务的请求
sudo ./grpcr \
  --input-raw="127.0.0.1:8080" \
  --output-stdout \
  --include-filter-method-match="^/UserService/.*$"
```

### 2. 限流控制

**场景**: 控制重放速率避免目标服务过载

```bash
# 限制每秒最多处理 100 个请求
sudo ./grpcr \
  --input-raw="127.0.0.1:8080" \
  --output-grpc="grpc://target-server:8080" \
  --rate-limit-qps=100
```

### 3. 多输出目标

**场景**: 同时输出到多个目标

```bash
# 同时输出到控制台、文件和另一个 gRPC 服务
sudo ./grpcr \
  --input-raw="127.0.0.1:8080" \
  --output-stdout \
  --output-file-directory="/tmp/backup" \
  --output-grpc="grpc://mirror-server:8080" \
  --record-response
```

### 4. 使用本地 Proto 文件

**场景**: 当目标服务没有启用反射时

```bash
# 指定单个 proto 文件
sudo ./grpcr \
  --input-raw="127.0.0.1:8080" \
  --output-stdout \
  --proto="./protos/service.proto"

# 指定 proto 文件目录
sudo ./grpcr \
  --input-raw="127.0.0.1:8080" \
  --output-stdout \
  --proto="./protos/"
```

### 5. RocketMQ 集成

**生产者模式** (将捕获的流量发送到 MQ):
```bash
sudo ./grpcr \
  --input-raw="127.0.0.1:8080" \
  --output-rocketmq-name-server="192.168.1.100:9876" \
  --output-rocketmq-topic="grpc-traffic" \
  --output-rocketmq-access-key="your-access-key" \
  --output-rocketmq-secret-key="your-secret-key"
```

**消费者模式** (从 MQ 读取流量并重放):
```bash
./grpcr \
  --input-rocketmq-name-server="192.168.1.100:9876" \
  --input-rocketmq-topic="grpc-traffic" \
  --input-rocketmq-group-name="replay-group" \
  --input-rocketmq-access-key="your-access-key" \
  --input-rocketmq-secret-key="your-secret-key" \
  --output-grpc="grpc://target-server:8080"
```

## 🏭 生产环境部署

### 1. 系统服务部署

**创建 systemd 服务文件** (`/etc/systemd/system/grpcreplay.service`):

```ini
[Unit]
Description=gRPC Replay Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/grpcreplay
ExecStart=/opt/grpcreplay/grpcr \
  --input-raw="0.0.0.0:8080" \
  --output-file-directory="/var/log/grpcreplay" \
  --output-file-max-size=500 \
  --output-file-max-backups=10 \
  --output-file-max-age=7 \
  --record-response
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

**启动服务**:
```bash
sudo systemctl daemon-reload
sudo systemctl enable grpcreplay
sudo systemctl start grpcreplay
sudo systemctl status grpcreplay
```

### 2. Docker 部署

**Dockerfile**:
```dockerfile
FROM golang:1.23-alpine AS builder

# 安装依赖
RUN apk add --no-cache libpcap-dev gcc musl-dev

WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=1 go build -o grpcr

FROM alpine:latest
RUN apk add --no-cache libpcap
COPY --from=builder /app/grpcr /usr/local/bin/
ENTRYPOINT ["grpcr"]
```

**运行容器**:
```bash
# 构建镜像
docker build -t grpcreplay .

# 运行 (需要特权模式进行网络捕获)
docker run --privileged --net=host \
  -v /tmp/grpc-logs:/logs \
  grpcreplay \
  --input-raw="0.0.0.0:8080" \
  --output-file-directory="/logs" \
  --record-response
```

### 3. Kubernetes 部署

**deployment.yaml**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grpcreplay
spec:
  replicas: 1
  selector:
    matchLabels:
      app: grpcreplay
  template:
    metadata:
      labels:
        app: grpcreplay
    spec:
      hostNetwork: true
      containers:
      - name: grpcreplay
        image: grpcreplay:latest
        securityContext:
          privileged: true
        args:
        - "--input-raw=0.0.0.0:8080"
        - "--output-file-directory=/logs"
        - "--record-response"
        volumeMounts:
        - name: logs
          mountPath: /logs
      volumes:
      - name: logs
        hostPath:
          path: /var/log/grpcreplay
```

## ⚡ 性能调优

### 1. 网络捕获优化

```bash
# 增加工作者数量提高并发处理能力
sudo ./grpcr \
  --input-raw="0.0.0.0:8080" \
  --output-grpc="grpc://target:8080" \
  --output-grpc-worker-number=10
```

### 2. 文件 I/O 优化

```bash
# 调整文件轮转参数
sudo ./grpcr \
  --input-raw="0.0.0.0:8080" \
  --output-file-directory="/fast-ssd/grpc-logs" \
  --output-file-max-size=1000 \
  --output-file-max-backups=5 \
  --output-file-max-age=1
```

### 3. 内存使用优化

```bash
# 设置合适的读取深度
./grpcr \
  --input-file-directory="/tmp/grpc-capture" \
  --input-file-read-depth=50 \
  --output-grpc="grpc://target:8080"
```

### 4. 系统级优化

**增加文件描述符限制**:
```bash
# 临时设置
ulimit -n 65536

# 永久设置 (/etc/security/limits.conf)
* soft nofile 65536
* hard nofile 65536
```

**网络缓冲区调优**:
```bash
# 增加网络缓冲区大小
echo 'net.core.rmem_max = 134217728' >> /etc/sysctl.conf
echo 'net.core.wmem_max = 134217728' >> /etc/sysctl.conf
sysctl -p
```

## 🔍 故障排查

### 1. 常见问题诊断

**权限问题**:
```bash
# 检查是否有 root 权限
sudo -v

# 检查 libpcap 是否正确安装
ldd ./grpcr | grep pcap
```

**网络接口问题**:
```bash
# 列出所有网络接口
ip addr show

# 检查端口是否被占用
netstat -tlnp | grep :8080
```

**gRPC 反射问题**:
```bash
# 测试 gRPC 反射是否可用
grpcurl -plaintext localhost:8080 list

# 使用本地 proto 文件替代反射
sudo ./grpcr \
  --input-raw="127.0.0.1:8080" \
  --output-stdout \
  --proto="./protos/"
```

### 2. 调试模式

**启用详细日志**:
```bash
# 设置日志级别为 debug
export SIMPLE_LOG_LEVEL=debug
sudo ./grpcr --input-raw="127.0.0.1:8080" --output-stdout
```

**性能监控**:
```bash
# 使用 pprof 进行性能分析
go tool pprof http://localhost:6060/debug/pprof/profile

# 监控资源使用
top -p $(pgrep grpcr)
```

### 3. 日志分析

**查看系统日志**:
```bash
# 查看 systemd 日志
journalctl -u grpcreplay -f

# 查看应用日志
tail -f /var/log/grpcreplay/*.log
```

## 📋 最佳实践

### 1. 安全实践

**最小权限原则**:
```bash
# 创建专用用户 (仍需 root 权限进行网络捕获)
sudo useradd -r -s /bin/false grpcreplay
sudo usermod -aG sudo grpcreplay
```

**数据脱敏**:
```bash
# 使用过滤器排除敏感接口
sudo ./grpcr \
  --input-raw="0.0.0.0:8080" \
  --output-stdout \
  --include-filter-method-match="^(?!.*/(Login|Password|Secret)).*$"
```

### 2. 监控和告警

**健康检查脚本**:
```bash
#!/bin/bash
# health-check.sh

PROCESS_COUNT=$(pgrep -c grpcr)
if [ $PROCESS_COUNT -eq 0 ]; then
    echo "grpcreplay is not running!"
    # 发送告警
    curl -X POST "https://hooks.slack.com/..." \
         -d '{"text":"grpcreplay service is down!"}'
    exit 1
fi

echo "grpcreplay is running normally"
```

**日志轮转配置** (`/etc/logrotate.d/grpcreplay`):
```
/var/log/grpcreplay/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 644 root root
    postrotate
        systemctl reload grpcreplay
    endscript
}
```

### 3. 容量规划

**存储容量估算**:
```
每个请求大小 ≈ 1-10KB (取决于 payload 大小)
QPS = 1000
每日数据量 ≈ 1000 * 5KB * 86400 ≈ 432GB/天
```

**网络带宽估算**:
```
原始流量: 100Mbps
镜像流量: 100Mbps (1:1 复制)
总带宽需求: 200Mbps
```

### 4. 测试策略

**渐进式部署**:
```bash
# 第一阶段：只记录，不转发
sudo ./grpcr \
  --input-raw="0.0.0.0:8080" \
  --output-file-directory="/tmp/test" \
  --rate-limit-qps=10

# 第二阶段：小流量转发
sudo ./grpcr \
  --input-raw="0.0.0.0:8080" \
  --output-grpc="grpc://test-server:8080" \
  --rate-limit-qps=100

# 第三阶段：全流量转发
sudo ./grpcr \
  --input-raw="0.0.0.0:8080" \
  --output-grpc="grpc://test-server:8080"
```

**A/B 测试**:
```bash
# 将 50% 流量发送到新版本服务
# (需要在应用层实现流量分割逻辑)
sudo ./grpcr \
  --input-raw="0.0.0.0:8080" \
  --output-grpc="grpc://service-v1:8080" \
  --output-grpc="grpc://service-v2:8080" \
  --include-filter-method-match=".*" \
  --rate-limit-qps=500
```

### 5. 运维自动化

**自动化部署脚本**:
```bash
#!/bin/bash
# deploy.sh

set -e

# 下载最新版本
wget https://github.com/vearne/grpcreplay/releases/latest/download/grpcr-linux-amd64.tar.gz
tar -xzf grpcr-linux-amd64.tar.gz

# 停止旧服务
sudo systemctl stop grpcreplay

# 备份旧版本
sudo cp /opt/grpcreplay/grpcr /opt/grpcreplay/grpcr.backup

# 安装新版本
sudo cp grpcr /opt/grpcreplay/
sudo chmod +x /opt/grpcreplay/grpcr

# 启动服务
sudo systemctl start grpcreplay

# 验证服务状态
sleep 5
sudo systemctl is-active grpcreplay || {
    echo "Service failed to start, rolling back..."
    sudo cp /opt/grpcreplay/grpcr.backup /opt/grpcreplay/grpcr
    sudo systemctl start grpcreplay
    exit 1
}

echo "Deployment successful!"
```

**监控脚本**:
```bash
#!/bin/bash
# monitor.sh

while true; do
    # 检查进程状态
    if ! pgrep -f grpcr > /dev/null; then
        echo "$(date): grpcreplay process not found, restarting..."
        sudo systemctl restart grpcreplay
    fi
    
    # 检查磁盘空间
    DISK_USAGE=$(df /var/log/grpcreplay | tail -1 | awk '{print $5}' | sed 's/%//')
    if [ $DISK_USAGE -gt 80 ]; then
        echo "$(date): Disk usage is ${DISK_USAGE}%, cleaning old files..."
        find /var/log/grpcreplay -name "*.log.gz" -mtime +7 -delete
    fi
    
    sleep 60
done
```

---

这些示例和最佳实践涵盖了从基础使用到生产环境部署的各个方面，帮助你更好地使用 grpcreplay 工具。根据具体需求选择合适的配置和部署方式。
