package socket

import (
	"ChatApp/dto"
	"ChatApp/errorConst"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

//当前服务收到 SIGINT/SIGTERM 直接退出，所有 ws 连接暴力销毁：
//前端瞬间全部断开，大规模同时发起重连，造成重连风暴
//生产环境需要：
//监听系统信号 → 遍历所有分片内全部连接，主动调用 conn.Close ()，给前端短暂窗口期平滑断开。
//开发环境可以暂时不加，上线前补充。
//Upgrader.CheckOrigin 生产环境禁止直接 return true 线上必须限制允许的 Origin 域名，防止跨域 websocket 劫持攻击

const ShardCount = 16

// WsInstance 饥饿初始化 在程序一启动就执行一次构造 后续复用同一个对象 不会重复初始化
var WsInstance = newWebSocketService()

type ConnectItem struct {
	MsgChan chan []byte // []byte 字节切片 用于存储聊天二进制/json消息
	Ctx     context.Context
	Cancel  context.CancelFunc
	closed  atomic.Bool // true = 连接下线 禁止写入消息 防止 panic 初始化时默认是 零值 false
}

type Shard struct {
	// 将锁内置到每个分片 每个分片各自使用一个锁 分片之间并行 分片之内串行
	mu sync.Mutex //互斥锁	用于保护 ConnMap 资源的访问	注意一定不要在函数内部去新建锁，否则每一个用户建立连接都有一个锁
	//另外这个sync.Mutex是非重入锁，不可连续加锁，需要注意解锁的时机
	connMap map[string]map[*websocket.Conn]*ConnectItem //分片连接池	当前分片根据对应用户 uid 获取相应连接 map，这样设计一个用户可以有多个连接，也就是多设备登录
}

// WebSocketService SocketService 字段小写为内部字段，外部仅可使用相应函数初始化服务
type WebSocketService struct {
	Upgrader *websocket.Upgrader // 升级器	用于将请求转换为websocket
	Shards   [ShardCount]*Shard  // 分片组数	长度为 ShardCount 存储类型为 *Shard的数组
}

// 获取用户对应的分片
func (wsi *WebSocketService) getShard(uid string) *Shard {
	h := fnv.New32a()             // 将 uid 字符串送入哈希函数 fnv，算出一个 32 位无符号整数哈希值
	_, _ = h.Write([]byte(uid))   // 同一个 uid 不管什么时候计算 哈希值永远一致 其下标永远不变 固定属于一个分片
	hashVal := h.Sum32()          // 得到一个 unit32 随机数字
	index := hashVal % ShardCount //哈希值对分片总数 例如 16 取模：结果一定落在 0 ~ 15
	return wsi.Shards[index]
}

func safeSendChan(ch chan []byte, data []byte) bool {
	defer func() {
		// recover 只有写在 defer 闭包内部才生效
		if err := recover(); err != nil {
			log.Println("safeSendChan err:", err) // 如果本函数你内部发生panic 捕获它消除崩溃
		}
	}()

	select {
	case ch <- data:
		return true
	default:
		// 此处消息超过缓冲区容量应该存入离线表或离线消息队列 暂时先搁置
		return false
	}
}

// newWebSocketService 导出指针类型可以让别的模块赋值引用
func newWebSocketService() *WebSocketService {
	wsi := &WebSocketService{
		Upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,                                       // 服务端读取前端消息的缓冲区大小 1024B = 1KB
			WriteBufferSize: 1024,                                       // 服务端向前端发送消息的缓冲区
			CheckOrigin:     func(r *http.Request) bool { return true }, // 跨域校验函数 true 表示允许跨域握手
		},
	}
	// 初始化所有分片
	for i := 0; i < ShardCount; i++ {
		wsi.Shards[i] = &Shard{
			connMap: make(map[string]map[*websocket.Conn]*ConnectItem),
		}
	}
	return wsi
}

// AddConn 注册用户的连接到连接池
func (wsi *WebSocketService) AddConn(uid string, conn *websocket.Conn, item *ConnectItem) {
	shard := wsi.getShard(uid) // 根据当前用户的 uid 为其分配到 对应的 分片辖区 shard
	shard.mu.Lock()            // 对该 shard 上锁 同一分片下的 用户 互相阻塞 不同分片不妨碍
	// 加锁之后要马上注册defer解锁，否则若两锁中间出现什么错误的话就无法解锁
	defer shard.mu.Unlock()

	// 注意这里必须要先检查用户map是否已经存在，否则每次用户建立连接会新建一个内部map覆盖原有的map，导致其他设备连接和消息丢失
	subMap, ok := shard.connMap[uid]
	if !ok {
		// 如果该用户未注册映射则注册映射
		subMap = make(map[*websocket.Conn]*ConnectItem)
		shard.connMap[uid] = subMap
	}

	// 初始化一条 ready 消息连接上之后立马发送给前端
	wc := &dto.WebsocketChange{
		MsgType: "ready",
		Data:    json.RawMessage(fmt.Sprintf("{\"uid\":\"%s\", \"conn\":\"%s\"}", uid, conn.RemoteAddr())),
	}
	msg, err := json.Marshal(wc)
	if err != nil {
		log.Printf("序列化消息错误: %v", err)
	}
	// 使用 safeSendChan 防止管道满时阻塞持有锁
	if !safeSendChan(item.MsgChan, msg) {
		log.Printf("ready 消息发送失败，管道已满")
	}

	// 将该连接注册到用户映射 并注入资源
	subMap[conn] = item
	// wsi.Mu.Unlock()	此处已解耦 在此函数作用域结束之后便会释放全局锁 后面两行注释为之前未解耦时的警告
	// 此处一定要手动解锁，而不是使用 defer ，defer 只会在当前函数退出再执行
	// 后续如果又有循环监听就会导致 该全局锁一直被某一个连接持有 别的用户将无法建立连接

	// 为当前连接启动独立消费者协程 否则其内部的持续 for 循环会让 connect 协程后续的 读 操作无法进行
	go wsi.ConsumeMessage(uid, conn, item)
}

// RemoveConn 清理用户断开的连接
func (wsi *WebSocketService) RemoveConn(uid string, conn *websocket.Conn) {
	shard := wsi.getShard(uid)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	// 当前用户是否注册映射
	subMap, exist := shard.connMap[uid]
	if !exist {
		return
	}
	// 当前连接是否注册映射
	item, existItem := subMap[conn]
	if !existItem {
		return
	}

	// 改用 ctx.Done 来控制消费协程的退出
	item.Cancel()
	// 标记失效 防止继续写入管道
	item.closed.Store(true)
	//close(item.MsgChan)
	// 删除连接映射
	delete(subMap, conn)

	// 当前用户无任何设备连接时 删除外层 key 释放内存
	if len(subMap) == 0 {
		delete(shard.connMap, uid)
	}
	//_ = conn.Close() 不再使用 等待上层读取循环出错自然关闭连接
}

// SendAllConnOfUid 将消息发送给指定用户的所有连接
func (wsi *WebSocketService) SendAllConnOfUid(uid string, wc *dto.WebsocketChange) error {
	shard := wsi.getShard(uid)
	shard.mu.Lock()
	subMap, exist := shard.connMap[uid]
	var items []*ConnectItem
	if exist {
		// 0：切片当前里面元素数量 = 0，len(items)=0
		// len(subMap)：预分配底层数组容量，提前开好一块内存，大小刚好能放下 subMap 所有元素
		items = make([]*ConnectItem, 0, len(subMap))
		// 此处无需判断 conn 的存在性 因为 nil map 只读不会 panic 而 写入则会 panic 此处是只读 nil map 的情况下直接循环 0 次
		// 为什么是 nil 因为 是 零值 如果 subMap 为 map[string]int 就是 0
		// key, value 此处只需返回对应的 value 即可 用不上 conn 这个 key
		for _, v := range subMap {
			items = append(items, v)
		}
	}
	// 锁内只收集快照，立刻释放
	shard.mu.Unlock()

	if len(items) == 0 {
		return errorConst.ErrUserOffline
	}

	msg, err := json.Marshal(wc)
	if err != nil {
		return err
	}

	// 注意 以上的快照 是 复制引用 指向还是同一个地址
	// 因此收集快照可能带来另外的问题 比如当快照收集完成并解锁之后 某个协程关闭了 MsgChan
	// 这就会导致后续的 item.MsgChan <- msg 触发 panic 此处先采用 recover 处理了 应该尽量避免使用 recover 处理 而是从架构上去处理并发的时序问题
	// 遍历目标用户的所有连接的管道并填入消息
	hasOnlineDevice := false
	for _, item := range items {
		// 失效连接直接跳过 避免 item.MsgChan <- msg 触发 panic
		if item.closed.Load() {
			continue
		}
		if safeSendChan(item.MsgChan, msg) {
			hasOnlineDevice = true
		} else {
			log.Printf("设备推送失败，管道已满或连接下线")
			//return errors.New("消息推送失败，管道已满或连接已下线")
			// 上方的 return 为错误写法 如果某个连接推送失败被 return，后续连接则全部收不到消息了
		}
	}
	// 快照在循环结束之后销毁引用 此时没有任何指针指向 ConnectItem 和 msgChan GC自动回收
	// 至少有一台设备投递成功 → 返回nil，不需要离线消息
	if hasOnlineDevice {
		return nil
	}

	// items >0，但是所有设备刚好在快照之后全部下线
	return errorConst.ErrUserOffline
}

// SendSingleConn 推送指定单条连接
func (wsi *WebSocketService) SendSingleConn(uid string, conn *websocket.Conn, wc *dto.WebsocketChange) error {
	shard := wsi.getShard(uid)
	shard.mu.Lock()
	subMap, userExist := shard.connMap[uid]
	if !userExist {
		shard.mu.Unlock() // 注意错误分支要在 return 前解锁
		return errorConst.ErrUserOffline
	}
	item, connExist := subMap[conn]
	shard.mu.Unlock()

	if !connExist {
		return errorConst.ErrConnOffline
	}

	if item.closed.Load() {
		return errorConst.ErrConnOffline
	}

	msg, err := json.Marshal(wc)
	if err != nil {
		return err
	}

	if !safeSendChan(item.MsgChan, msg) {
		return errorConst.ErrSendFailed
	}

	return nil
}

// ConsumeMessage 消费该连接产生的数据
func (wsi *WebSocketService) ConsumeMessage(uid string, conn *websocket.Conn, item *ConnectItem) {
	for {
		select {
		case <-item.Ctx.Done():
			log.Printf("用户：%s 连接 %v 上下文取消，退出消费者协程", uid, conn.RemoteAddr().String())
			return
		case msg := <-item.MsgChan:
			// *websocket.Conn 规定：同一时间只能有 1 个 协程 调用 WriteMessage
			// 多协程并发写会直接报文错乱、连接崩溃
			// 比如说在同一时间内有两个网络请求 此时 gin/go原生http服务 都会为这两个请求 开启了两个独立的 协程
			// A协程 使用了 conn.WriteMessage 发送好友申请
			// B协程 使用了 相同的 conn 要进行 WriteMessage 发送聊天消息
			// 就会变成并发写入，就像是两个人同时对同一张纸张进行书写，两人的字都叠在一起，根本看不清写的是什么
			// 因此需要使用一个管道来缓存消息，并在此处同步执行写操作
			err := conn.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				log.Printf("用户：%s 发送消息失败，清理连接: %v", uid, err)
				// 发送失败，清理断开的连接
				wsi.RemoveConn(uid, conn)
				return
			}
		}
	}
}

// NewMsgContext 为每一条前端发送的消息创建独立业务 context
func NewMsgContext(connCtx context.Context) (context.Context, context.CancelFunc) {
	msgCtx, msgCancel := context.WithTimeout(connCtx, time.Second*5)
	return msgCtx, msgCancel
}

// IsUserOnline 目标用户是否在线
func (wsi *WebSocketService) IsUserOnline(uid string) bool {
	shard := wsi.getShard(uid)
	shard.mu.Lock()
	_, userExist := shard.connMap[uid]
	if !userExist {
		shard.mu.Unlock() // 注意错误分支要在 return 前解锁
		return false
	}
	shard.mu.Unlock()
	return true
}
