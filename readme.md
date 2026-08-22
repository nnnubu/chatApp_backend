# ChatApp Backend

基于 Go + Gin + GORM 的即时通讯后端服务，支持私聊、好友系统、WebSocket 实时消息推送、游标分页历史消息、消息已读回执等核心 IM 功能。采用 Docker Compose 一键部署，本地交叉编译二进制上传服务器运行。

## 技术栈

| 类别 | 技术 | 版本 | 在项目中的作用 |
|---|---|---|---|
| 语言 | Go | 1.22+ | 后端主语言 |
| Web 框架 | Gin | — | HTTP 路由、中间件、静态资源 |
| ORM | GORM | — | 数据库操作、事务、自动迁移 |
| 数据库 | MySQL | 8.0 | 用户、好友、消息、会话数据持久化 |
| 缓存 | Redis | 7 | 预留缓存层（目前就发送验证码用上了 后续也可做对高频访问的资源进行缓存的功能） |
| 实时通信 | gorilla/websocket | — | WebSocket 连接管理、消息推送 |
| 鉴权 | JWT | — | 登录态签发与校验（HTTP + WS 共用） |
| 消息 ID | 雪花算法 | — | 消息主键，趋势递增，支持游标分页 |
| 用户 ID | UUID v7 | — | 用户对外标识，时间有序 |
| 日志 | zap | — | 结构化日志 |
| 部署 | Docker Compose | — | MySQL + Redis + 后端一键编排 |

## 系统架构图

```
┌─────────────────────────────────────────────────────────────────────┐
│                        客户端 (Flutter APP)                          │
│   ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│   │  HTTP 请求    │  │ WebSocket    │  │  本地存储 SharedPrefs     │  │
│   │  (Dio)       │  │ 长连接        │  │  (登录态/用户信息)        │  │
│   └──────┬───────┘  └──────┬───────┘  └──────────────────────────┘  │
└──────────┼─────────────────┼────────────────────────────────────────┘
           │ /api/* /auth/*  │ /ws/connect (JWT in Header)
           ▼                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     Gin 服务 (容器内 :8080)                          │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  中间件层                                                     │   │
│  │  CORS  │  JWT鉴权  │  请求超时Context注入  │  UA反爬  │ 静态缓存 │   │
│  └──────────────────────────────────────────────────────────────┘   │
│  ┌─────────────────┐  ┌─────────────────────────────────────────┐   │
│  │  HTTP 路由层     │  │  WebSocket 层                           │   │
│  │  /api  公开接口  │  │  ┌───────────────────────────────────┐  │   │
│  │  /auth 鉴权接口  │  │  │  分片连接池 (16 Shards)            │  │   │
│  │                 │  │  │  Shard[0..15] 每片独立锁           │  │   │
│  │  handler/       │  │  │    └─ uid → {conn1, conn2, ...}   │  │   │
│  │    user/        │  │  │  每连接: MsgChan + 消费协程         │  │   │
│  │    friend/      │  │  │  消息推送: 锁内快照 → 锁外广播       │  │   │
│  │    message/     │  │  └───────────────────────────────────┘  │   │
│  │    msgCategory/ │  │  socket/service/                        │   │
│  └────────┬────────┘  │    connect.go  createMessage.go         │   │
│           │           │    pushWsMsg.go  replayAck.go           │   │
│           ▼           └─────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  业务逻辑层 service/  →  model/ (数据操作 + 事务)              │   │
│  └──────────────────────────────────────────────────────────────┘   │
└──────────┬───────────────────────────────┬──────────────────────────┘
           │                               │
           ▼                               ▼
┌─────────────────────┐      ┌──────────────────────────────────────┐
│   MySQL 8.0 容器     │      │          Redis 7 容器                │
│  (持久化命名卷)       │      │       (虽然当前只用到了存放验证码)      │
│                     │      │                                      │
│  users              │      │  预留: 在线状态 / 会话缓存 / 限流       │
│  friend_applies     │      │                                      │
│  friends            │      │                                      │
│  messages           │      │                                      │
│  conversations      │      │                                      │
│  conversation_members│     │                                      │
│  conversation_reads │      │                                      │
│  msg_categories     │      │                                      │
│  verify_codes       │      │                                      │
└─────────────────────┘      └──────────────────────────────────────┘
```

## 数据库 ER 图

```
┌──────────────┐         ┌──────────────────┐         ┌──────────────┐
│    users     │         │  friend_applies  │         │   friends    │
│──────────────│         │──────────────────│         │──────────────│
│ id (PK,自增)  │         │ id (PK,自增)      │         │ id (PK,自增)  │
│ uid (唯一)    │◄────────│ apply_uid (FK)   │    ┌───►│ uid (FK)     │
│ nickname     │    ┌───►│ target_uid (FK)  │───┘    │ friend_uid(FK)│
│ password     │    │    │ msg_id (唯一)     │         │ remark       │
│ email (唯一)  │    │    │ msg              │         │ status       │
│ avatar       │    │    │ status           │         │ created_at   │
│ bg_img       │    │    │ apply_read       │         └──────────────┘
│ intro        │    │    │ created_at       │
│ birthday     │    │    └──────────────────┘
│ created_at   │    │
│ updated_at   │    │
└──────┬───────┘    │
       │            │
       │            │
       ▼            ▼
┌──────────────────────────────────────────────────────────────────┐
│                        消息与会话模型                              │
│                                                                  │
│  ┌─────────────────┐    ┌────────────────────┐                    │
│  │  conversations  │    │ conversation_members│                    │
│  │─────────────────│    │────────────────────│                    │
│  │ conversation_uid│◄──►│ conversation_uid   │──┐                 │
│  │ type(1私聊/2群聊)│    │ uid (FK→users)     │  │                 │
│  │ created_at      │    └────────────────────┘  │                 │
│  └────────┬────────┘                            │                 │
│           │                                     │                 │
│           ▼                                     ▼                 │
│  ┌─────────────────┐              ┌─────────────────────────┐     │
│  │    messages     │              │   conversation_reads     │     │
│  │─────────────────│              │─────────────────────────│     │
│  │ msg_id (PK,雪花) │◄────────────│ uid (FK,联合主键)         │     │
│  │ sender_uid (FK) │              │ conversation_uid(联合主键)│     │
│  │ receiver_uid(FK)│              │ last_read_msg_id (水位线) │     │
│  │ conversation_uid│              │ updated_at               │     │
│  │ msg_type        │              └─────────────────────────┘     │
│  │ content         │                                                │
│  │ created_at      │                                                │
│  └─────────────────┘  索引: (conversation_uid, msg_id)             │
│                                                                  │
│  设计要点:                                                         │
│  · 已读不逐条记录, 用会话级水位线 last_read_msg_id, O(1) 更新       │
│  · 消息 msg_id 雪花算法趋势递增, 天然支持游标分页                    │
│  · 会话与成员分离, 私聊/群聊统一模型, 扩展群聊只需加类型             │
└──────────────────────────────────────────────────────────────────┘
```

## 核心消息链路

### 一、消息发送完整链路（前端 → 后端 → 双方推送）

```
前端 Flutter                          后端 Go (Gin + WebSocket)
─────────────                         ───────────────────────────
用户输入消息, 点击发送
      │
      ▼
WebSocketService.sendDto(chatDto)
      │  生成 requestId (nanoid)
      ▼
加入发送队列 _outGoingQueue (FIFO)
      │
      ▼
队列消费者 _startConsumeOutGoing()
  ├─ 检查: 通道已连接? 后端ready? 链路健康?
  ├─ 不满足 → 暂停消费, 保留队列
  └─ 满足 → 继续
      │
      ▼
AckHelper.addPendingId(dto)
  ├─ 注册 requestId → AckListener
  ├─ 启动 8s 超时定时器
  └─ 超过 3 次重试 → 消息流放(roamed)
      │
      ▼
WebsocketConnector.send(dto)
  └─ JSON 编码 → WebSocket 文本帧发送
      │
      │  (网络传输)
      ▼
后端 connect.go 读循环 conn.ReadMessage()
  ├─ 重置 30s 读超时
  ├─ 解析 WebsocketChange {msgType, requestId, data}
  └─ msgType == "chat" → 进入聊天处理
      │
      ▼
解析 ChatReq {receiverUID, conversationUID, content}
  ├─ 校验: 不能发给自己
  └─ 生成 newMsgId (雪花算法)
      │
      ▼
CreateMessage() —— 数据库事务
  ├─ GetPrivateConversation(): 子查询+分组, 校验双方私聊会话合法性
  ├─ 会话不存在 → 返回错误(当前仅好友可私聊)
  ├─ 会话存在但 conversationUID 不匹配 → 拒绝
  └─ CreateMessage(): 消息入库 messages 表
      │
      ▼  事务提交成功
      │
      ├──────────────────────────────────────┐
      ▼                                      ▼
同步回复 ACK                          异步协程推送消息
ReplayAck(uid, conn, requestId,       go func() {
  newMsgId, success=true)                查询发送者/接收者用户信息
      │                                   PushBroadCastMsg(发送者)
      │  (单连接推送)                       PushBroadCastMsg(接收者)
      ▼                                }
前端收到 ack                          SendAllConnOfUid(targetUid)
AckHelper.onReciveAck()                 ├─ getShard(uid) → 分片锁
  ├─ 移除 pending 记录                   ├─ 锁内收集该用户所有连接快照
  ├─ 取消超时定时器                      ├─ 释放锁
  └─ 推送 AckStatus.success 事件         └─ 锁外遍历快照:
      │                                    safeSendChan(MsgChan, msg)
      ▼                                  每条连接的独立消费协程:
UI: 消息标记为"已送达"                   ConsumeMessage()
                                        └─ conn.WriteMessage() 串行写入
                                              │
                                              ▼
                                        前端双方各自收到 chat 消息
                                        MessageDispatcher.dispatch()
                                              │
                                              ▼
                                        ChatHandler.handle()
                                              │
                                              ▼
                                        广播 MessageListEvent
                                        UI: 聊天列表插入新消息
```

### 二、消息接收与分发流程（后端推送 → 前端处理）

```
后端 WebSocket 层                         前端 Flutter
───────────────                           ─────────────
PushBroadCastMsg(targetUid, "chat", msgId, payload)
      │
      ▼
SendAllConnOfUid(targetUid, wc)
      │
      ├─ 1. fnv32(uid) % 16 → 定位分片 Shard
      ├─ 2. shard.mu.Lock()  加分片锁
      ├─ 3. 读取 subMap = connMap[uid]
      │     收集 []*ConnectItem 快照(复制引用)
      ├─ 4. shard.mu.Unlock()  立即释放锁
      │     (不在持锁状态下做网络IO)
      └─ 5. 遍历快照:
            if item.closed → 跳过
            safeSendChan(item.MsgChan, msg)
              ├─ chan 有空间 → 写入成功
              └─ chan 已满 → 不实时推送等待拉取离线消息
      │
      ▼  (每条连接独立的消费协程, 与读循环并发)
ConsumeMessage(uid, conn, item)
  for {
    select {
      case <-item.Ctx.Done(): return
      case msg := <-item.MsgChan:
        conn.WriteMessage(TextMessage, msg)
          │  (串行写入, 避免多协程并发写导致报文错乱)
          └─ 失败 → RemoveConn(uid, conn) 清理连接
    }
  }
      │
      │  (网络传输)
      ▼
WebsocketConnector._channel.stream.listen(rawData)
      │
      ├─ JSON 解码 → MessageDto {msgType, msgId, requestId, data}
      └─ 抛出 MessageEvent(dto)
      │
      ▼
WebSocketService 事件监听:
  ├─ msgType == ready     → 后端就绪, 启动心跳, 拉离线消息
  ├─ msgType == heartBeat → 重置心跳计数(不进分发器)
  ├─ msgType == ack       → AckHelper.onReciveAck()(不进分发器)
  └─ 其他业务消息走分发器
      │
      ▼
MessageDispatcher.instance.dispatch(dto)
      │
      ├─ _handlerMap[dto.msgType] 查找注册的处理器
      ├─ 未注册 → 打印日志, 丢弃
      └─ 找到 handler →
          │
          ▼
      handler.handle(dto)  (异步)
          │
          ├─ ChatHandler: 解析 data → 构建消息卡片 → MessageListEvent
          ├─ FriendApplyHandler: 解析好友申请 → MessageListEvent
          └─ CategoryPullHandler: 解析分类 → CategoryListEvent
          │
          ▼
      _eventBus.add(event)  广播到消息总线
          │
          ▼
      各页面/控制器订阅 eventBus
      UI 更新: 聊天页插入消息 / 首页更新申请列表
```

### 三、WebSocket 连接建立与健康检测

```
前端                                    后端
────                                    ────
WebSocketService.connect(token)
      │
      ▼
WebsocketConnector.connect(url, token)
  ├─ IOWebSocketChannel.connect(headers: Authorization)
  └─ 标记 status = connected (仅代表发起连接, 不代表握手成功)
      │
      │  (HTTP Upgrade 请求, 携带 JWT)
      ▼
Gin 路由 /ws/connect
  ├─ AuthMiddleware: 校验 JWT, 注入 uid
  └─ WebSocketConnect(c)
      │
      ▼
Upgrader.Upgrade() → 协议升级 101 Switching Protocols
      │
      ▼
AddConn(uid, conn, item)
  ├─ getShard(uid) → 分片锁
  ├─ 注册 uid → {conn: ConnectItem{MsgChan, Ctx, Cancel}}
  ├─ 发送 ready 消息 {uid, conn地址}
  ├─ 启动该连接的消费协程 ConsumeMessage()
  └─ 进入读循环 for { ReadMessage() }
      │
      ▼
前端收到 ready
  ├─ _backendReady = true
  ├─ 取消 5s ready 超时定时器
  ├─ HeartBeat.start() 启动心跳
  └─ 唤醒发送队列, 拉取离线消息
      │
      ▼
心跳机制 (HeartBeat):
  ├─ 每 10s 发送 ping
  ├─ 启动 6s pong 等待定时器
  ├─ 收到任意后端消息 → resetHeartBeat() (健康, 清零丢失计数)
  └─ pong 超时 → _lostPongCount++
       └─ 连续丢失 3 次 → markDead()
            └─ onConnectDead 回调 → _forceCloseConnection()
      │
      ▼
连接断开/异常:
  ├─ _allowReconnect == true → _reconnect()
  ├─ 指数退避: 2s → 4s → 8s → 16s (上限)
  ├─ 最多 5 次自动重连
  ├─ 耗尽 → autoReconnectExhausted = true (UI 显示手动重连)
  └─ 心跳健康(healthy)时重置: 退避回 2s, 次数回 5
      │
      ▼
重连成功 → 收到 ready → 拉取离线消息 → 补发队列中积压的消息
```

### 四、ACK 可靠传输机制

```
发送方                                    接收方/服务端
──────                                    ───────────
sendDto(dto)
  │  dto.requestId = nanoid()
  ▼
AckHelper.addPendingId(dto)
  ├─ _pending[requestId] = AckListener{
  │    timer: 8s 后触发 onTimeOut,
  │    dto: 原始消息(失败时用于重发),
  │    status: pending,
  │    requestCount: 1
  │  }
  ▼
发送消息 → ... → 后端处理
                                        后端处理完成
                                        ReplayAck(requestId, msgId, success)
                                          │  通过当前连接单播 ack
                                          ▼
                                  前端收到 ack 消息
                                  AckHelper.onReciveAck(dto)
                                    ├─ _pending 中找到 requestId
                                    ├─ data.success == true:
                                    │   ├─ 移除 pending 记录
                                    │   ├─ cancel 定时器
                                    │   ├─ 用后端返回的 msgId 更新 dto
                                    │   └─ 推送 AckStatus.success
                                    └─ data.success == false:
                                        └─ 打印 errMsg(不自动重试, 等超时统一处理)
  │
  │  超时分支(8s 内未收到 ack):
  ▼
AckHelper.onTimeOut(requestId)
  ├─ listener.status = failed
  └─ 推送 AckStatus.failed
      │
      ▼
WebSocketService 监听 ackRespStream:
  ├─ failed → 将原始 dto 重新加入 _outGoingQueue
  │   → 队列消费者再次发送
  │   → addPendingId 时 requestCount++ (第2次, 第3次)
  ├─ 第3次仍超时 → addPendingId 判定超限
  │   → 移除 pending, 加入 roamedMsgList
  │   → 推送 AckStatus.roamed (消息流放, UI 可提示重发，被流放的消息暂时还没做用户手动重复功能)
  └─ success → 正常消费下一条
```

### 五、分片连接池并发架构

```
                    WebSocketService (单例)
                    ┌─────────────────────────────────────┐
                    │  Shards[0..15]  (16 个分片)          │
                    │                                     │
                    │  Shard[0]          Shard[1]   ...   │
                    │  ┌──────────┐     ┌──────────┐     │
                    │  │ mu Mutex │     │ mu Mutex │     │
                    │  │ connMap  │     │ connMap  │     │
                    │  └────┬─────┘     └────┬─────┘     │
                    └───────┼────────────────┼───────────┘
                            │                │
              fnv32("uidA")%16=3             │
                            │                │
                            ▼                ▼
                    Shard[3]                 ...
                    ┌──────────────────────────────┐
                    │ mu.Lock()                     │
                    │ connMap:                      │
                    │   "uidA": {                   │
                    │     connA1: ConnectItem{      │
                    │       MsgChan(chan, buffer100)│
                    │       Ctx, Cancel             │
                    │       closed: atomic.Bool     │
                    │     },                        │
                    │     connA2: ConnectItem{...}  │  ← 多设备登录
                    │   },                          │
                    │   "uidB": { connB1: {...} }   │
                    │ mu.Unlock()                   │
                    └──────────────────────────────┘

并发安全设计:
  · 不同 uid 哈希到不同分片 → 完全并行, 无锁竞争
  · 同一分片内的 uid 操作 → 串行(分片锁保护)
  · 消息推送: 锁内只做快照收集(读), 锁外做网络IO(写)
  · 连接关闭: atomic.Bool 标记 closed, 写入前检查, 防止 panic
  · 每连接独立 MsgChan + 单消费协程 → 串行 WriteMessage, 避免并发写错乱
```

## 已实现功能

### 用户模块
- 邮箱注册 / 登录 / 重置密码
- 邮箱验证码发送（SMTP）
- 头像 / 背景图上传（自动压缩裁剪）
- 个人资料修改
- 个人二维码生成（扫码加好友）
- 陌生人资料预览

### 好友模块
- 发起好友申请（附带验证消息）
- 同意 / 拒绝好友申请
- 重复申请拦截（待处理状态不可重复发送）
- 好友列表分页拉取
- 离线好友申请拉取（登录后同步未处理的申请）
- 申请处理结果回执（申请人可收到同意/拒绝通知）

### 消息模块
- 私聊会话自动创建（首次发消息时建立会话）
- WebSocket 实时消息推送
- 游标分页拉取历史消息（基于雪花 ID 游标）
- 未读消息拉取（登录后同步所有会话的离线消息）
- 会话级已读回执（水位线设计）
- 多设备同时在线（消息广播到所有设备）
- 消息 ACK 确认（8s 超时，最多重试 3 次）

### WebSocket 架构
- **分片连接池**：16 分片 + 分片锁，降低锁竞争
- **多设备支持**：`uid → [conn1, conn2, ...]` 映射
- **消费者协程模型**：每连接独立 `MsgChan` + 消费协程，串行写入
- **并发安全**：`atomic.Bool` 标记失效，锁内快照 + 锁外发送
- **心跳检测**：10s 周期 ping，6s pong 超时，连续丢 3 次判定死亡
- **指数退避重连**：2s→4s→8s→16s，最多 5 次，健康后重置
- **发送队列**：FIFO 队列，连接未就绪时积压，恢复后自动补发

## 项目结构

```
chatApp_backend/
├── config/              # 配置加载与模板
│   ├── config.go        # 配置结构体与加载逻辑
│   └── config.json.template
├── database/            # 数据库连接（MySQL / Redis，含重试机制）
├── dto/                 # 数据传输对象
├── errorConst/          # 统一错误码
├── global/              # 全局变量（日志等）
├── handler/             # HTTP 接口处理层
│   ├── user/  friend/  message/  msgCategory/
├── middleware/          # JWT 鉴权中间件
├── model/               # 数据模型与数据库操作
│   ├── user.go  friend.go  message.go  msgCategory.go
│   ├── verifyCode.go  commonResp.go
├── routers/             # 路由注册
├── service/             # 业务逻辑层
├── socket/              # WebSocket 核心
│   ├── socketMain.go    # 分片连接池、连接管理、消息推送
│   └── service/         # WS 业务处理
│       ├── connect.go        # WS 连接入口 + 读循环 + 消息分发
│       ├── createMessage.go  # 消息入库 + 异步推送
│       ├── pushWsMsg.go      # 广播/单播推送封装
│       ├── replayAck.go      # ACK 回执
│       └── markReadFriendApply.go
├── static/              # 静态资源
├── utils/               # 工具函数（雪花算法、JWT、加密等）
├── docker/Dockerfile
├── docker-compose.yml
├── main.go
├── go.mod / go.sum
└── README.md
```

## 接口概览

### 公开接口（无需鉴权）
| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/login` | 登录 |
| POST | `/api/register` | 注册 |
| POST | `/api/resetPwd` | 重置密码 |
| POST | `/api/sendCode` | 发送邮箱验证码 |

### 鉴权接口（需 JWT）
| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/auth/uploadImage` | 上传图片 |
| POST | `/auth/updateInfo` | 修改个人资料 |
| POST | `/auth/getUserQR` | 获取个人二维码 |
| POST | `/auth/visitOthers` | 查看陌生人资料 |
| POST | `/auth/addFriend` | 发起好友申请 |
| POST | `/auth/updateApply` | 同意/拒绝好友申请 |
| GET | `/auth/pullOfflineApply` | 拉取离线好友申请 |
| GET | `/auth/pullCategory` | 拉取消息分类列表 |
| GET | `/auth/pullFriends` | 拉取好友列表 |
| GET | `/auth/pullHistoryMessage` | 游标分页拉取历史消息 |
| GET | `/auth/pullUnReadMessage` | 拉取所有未读消息 |
| POST | `/auth/markReadStatus` | 标记会话已读 |

### WebSocket 消息类型
| 类型 | 方向 | 说明 |
|---|---|---|
| `ready` | 后端→前端 | 连接建立成功 |
| `ping` / `pong` | 双向 | 心跳保活 |
| `chat` | 双向 | 聊天消息 |
| `addFriend` | 后端→前端 | 好友申请通知 |
| `argee` / `refuse` | 后端→前端 | 好友申请同意/拒绝回执 |
| `markRead` | 前端→后端 | 标记已读 |
| `ack` | 后端→前端 | 消息送达确认 |
| `pullCategory` | 双向 | 拉取消息分类 |

## 本地开发

### 环境要求
- Go 1.22+
- MySQL 8.0
- Redis 7

### 步骤
```bash
git clone https://github.com/nnnubu/chatApp_backend.git
cd chatApp_backend
cp config/config.json.template config/config.json
# 编辑 config.json，填入本地 MySQL/Redis 连接信息
go mod tidy
go run main.go
```

服务默认监听 `8080` 端口。

## Docker 部署

> 采用**本地交叉编译二进制 + 服务器 Docker 运行**，不在服务器上编译 Go 代码。

### 1. 本地交叉编译（Windows PowerShell）
```powershell
$env:CGO_ENABLED=0; $env:GOOS="linux"; $env:GOARCH="amd64"
go build -o chatapp main.go
```

### 2. 上传到服务器
```powershell
scp .\chatapp root@你的服务器IP:/opt/chatapp/
```

### 3. 服务器启动
```bash
cd /opt/chatapp
# 确保 config/configDocker.json 已配置真实密码
docker compose up -d --build
docker compose logs -f chatapp
```

### 包含的服务
- `chatapp`：后端（700m 内存 / 1.2 CPU）
- `mysql`：MySQL 8.0（600m 内存，数据持久化）
- `redis`：Redis 7 Alpine（256m 内存，缓存）

## 配置说明

`config/config.json`：
```json
{
  "mysql": {"username":"root","password":"你的密码","host":"mysql","port":"3306"},
  "redis": {"addr":"redis:6379","password":"","db":0},
  "smtp":  {"account":"邮箱","password":"授权码","host":"smtp.qq.com","port":25},
  "jwt":   {"secret":"你的JWT密钥","expire":"168h"},
  "app":   {"base-path":"./static","pre-fix-url":"http://0.0.0.0:8080"},
  "ImageResize": {"avatar_w":256,"avatar_h":256,"bg_w":720,"bg_h":405}
}
```

> Docker 环境下 `mysql.host` 填服务名 `mysql`，`redis.addr` 填 `redis:6379`。

## 项目难点与设计思考

1. **WebSocket 并发写安全**：同一连接多协程 `WriteMessage` 会报文错乱，采用每连接独立管道 + 单消费协程串行写入
2. **分片锁降低竞争**：全局单锁高并发下成瓶颈，按 UID 哈希分 16 片，分片间并行
3. **锁内快照 + 锁外发送**：推送时持锁只收集连接引用快照，释放锁后再做网络 IO，平衡安全与性能
4. **已读水位线优化**：传统逐条已读记录随消息量爆炸，改用会话级 `last_read_msg_id`，O(1) 更新
5. **游标分页**：雪花 ID 趋势递增，用 `msg_id < cursor` 替代 OFFSET，避免深分页性能恶化
6. **容器启动时序**：`depends_on` 只保证容器启动顺序不保证服务就绪，在 DB 连接层加重试（10 次 × 2s）
7. **静默断网检测**：TCP 重传超时前无法感知断连，用心跳 ping/pong + 连续丢失阈值主动判定死亡
8. **半开放连接**：`IOWebSocketChannel.connect` 同步返回不代表握手成功，需等待后端 `ready` 信号才允许业务通信
9. **指数退避重连**：断网瞬间大量重连会造成重连风暴，用 2→4→8→16s 退避，健康后重置

## 待开发

- [ ] 群聊功能
- [ ] 消息撤回
- [ ] 消息搜索
- [ ] 图片/文件/语音消息
- [ ] 读书模块
- [ ] WebSocket 平滑下线
- [ ] CheckOrigin 域名限制

## 相关仓库

- 前端 Flutter：[chatApp_frontend](https://github.com/nnnubu/chatApp_frontend)
