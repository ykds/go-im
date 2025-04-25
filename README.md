## Introduction
go-im 是基于 Golang + Gin + Gorm 框架实现的即使聊天服务，主要服务与功能：
- api服务
- 用户服务
- 消息服务
- 发号器服务
- 接入层

**api服务**调用/编排下层用户服务、消息服务为前端提供业务接口访问。

**用户服务**提供与用户、好友相关的业务处理，如注册、登录、个人信息、好友申请、好友列表等功能。

**消息服务**提供与包含消息、会话、群组模块的功能，如发送消息、获取聊天会话、创建与获取群组列表等功能。

**发号器服务**用于在发消息时获取会话级别的唯一消息ID，用于保证消息的有序、不丢、不重复。

**接入层服务**前端通过连接接入层，并通过心跳维持在线状态，为前端提供实时的消息通知。

## Features
- Etcd 实现微服务注册与发现
- OpenTelemtry + Jaeger实现微服务之间链路追踪
- Prometheus + Grafana 实现应用指标上传与监控
- ELK 实现分布式日志收集
- ackqueue 实现消息确认和重传
- msgbox 实现会话级别消息信箱
- kafka 支撑高吞吐消息发送

## Arch
![arch](./doc/arch.png)

api服务、用户服务、消息服务、接入层、发号器服务都是以微服务运行，并通过 etcd 完成服务发现。

api服务、用户服务、消息服务之间通过 grpc 来实现通讯。

接入层与用户服务、消息服务通过 kafka 来实现通知和消息的实时前端推送。



## Deployment
```
Linux 环境：
- Ubuntu 22.04
- MySQL 8.0
- Docker 28.0.4
- Docker Compose 2.34.0
- Go 1.23.4
```

1. 创建数据表
> CREATE DATABASE im;
> mysql goim < goim.sql

2. Docker 运行 `Kafka、Redis、Jaeger、etcd`
   
修改`.env`中`HOST`为主机提供外部访问IP地址

如虚拟机IP：
> HOST=192.168.x.x

运行 Docker 容器
> docker compose -f kafka.yaml up -d
>
> docker compose -f other.yaml up -d
>
> > 如果不需要 Jaeger 和 Etcd，将上一条命令换成：
>
> docker compose -f other.yaml up redis -d

4. Docker 运行 `promethues、grafana` (可选)
> docker compose -f promethues.yaml up -d

3. 运行 `ELK` (可选)
> docker compose -f elk.yaml up -d

4. 编译服务
> make build-all

5. 重命名`cmd/*`下每个服务的`config.yaml.example`为`config.yaml`并修改配置

6. 启动服务
> ./build/gateway -c cmd/gateway/config.yaml
> 
> ./build/user -c cmd/user/config.yaml
> 
> ./build/message -c cmd/message/config.yaml
> 
> ./build/access -c cmd/access/config.yaml
> 
> ./build/seqserver -c cmd/seqserver/config.yaml

## Demo演示
   - ### 注册
  ![arch](./doc/reg.png)
   - ### 登录
  ![arch](./doc/login.png)
   - ### 聊天会话
  ![arch](./doc/chat1.png)
  ![arch](./doc/chat2.png)
   - ### 个人信息
  ![arch](./doc/info.png)

