## 概述
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


## 架构图
![arch](./doc/arch.png)

api服务、用户服务、消息服务、接入层、发号器服务都是以微服务运行，并通过 etcd 完成服务发现。

api服务、用户服务、消息服务之间通过 grpc 来实现通讯。

接入层与用户服务、消息服务通过 kafka 来实现通知和消息的实时前端推送。



## 部署
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
> mysql im < im.sql

2. Docker 运行 Kafka、Redis、Jaeger、etcd
   
修改`.env`中`HOST`为主机提供外部访问IP地址
> docker compose -f kafka.yaml up -d
>
> docker compose -f other.yaml up -d

3. 运行 ELK (可选)
> docker compose -f elk.yaml up -d

4. 编译服务
> make build-all

5. 启动服务
> ./build/api_gateway
> 
> ./build/user
> 
> ./build/message
> 
> ./build/access
> 
> ./build/seqserver

## Demo演示
   - 注册
   - 登录
   - 添加好友
   - 发送消息
