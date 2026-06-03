package market

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// ==================== WebSocket 升级器 ====================

// Upgrader 将 HTTP 连接升级为 WebSocket 连接
var Upgrader = &websocket.Upgrader{
	// CheckOrigin：跨域检查。开发阶段允许所有来源，生产环境要限制
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	// 读写缓冲区大小（单位：字节）
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// ==================== 客户端 ====================

// Client 代表一个 WebSocket 客户端连接
type Client struct {
	hub    *Hub            // 所属的 Hub
	conn   *websocket.Conn // WebSocket 连接
	send   chan []byte     // 发送消息的缓冲通道
	userID uint            // ← 新增：该连接对应的用户ID（0 表示未登录）
	connID string          // ← 新增：连接唯一标识
}

// readPump 从 WebSocket 连接读取消息（本项目中客户端只订阅，不发消息）
// 这个方法在独立的 goroutine 中运行
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c // 连接关闭时从 Hub 注销
		c.conn.Close()        // 关闭 WebSocket 连接
	}()

	for {
		_, _, err := c.conn.ReadMessage() // 读取消息（但我们不处理它）
		if err != nil {
			// 连接关闭或发生错误，退出循环
			break
		}
		// 不做读取处理，客户端只接收行情推送
	}
}

// writePump 向 WebSocket 连接写入消息
// 这个方法在独立的 goroutine 中运行
func (c *Client) writePump() {
	defer c.conn.Close()

	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			// 写入失败（客户端断开），退出循环，send 通道会被 Hub 清理
			return
		}
	}
}

// ==================== Hub ====================

// Hub 管理所有 WebSocket 客户端连接
// 这是整个实时推送系统的核心
type Hub struct {
	// clients 按交易对分组存储客户端
	// key: 交易对（如 BTC-USDT）, value: 该交易对的订阅者集合
	// sync.RWMutex 读写锁，读多写少场景性能优于 sync.Mutex
	mu      sync.RWMutex
	clients map[string]map[*Client]bool // 交易对 -> 客户端集合

	// ===== 新增：按用户ID索引 =====
	// key: userID, value: 该用户的所有 WebSocket 连接（一个用户可能打开多个页面）
	userClients map[uint]map[*Client]bool

	// 注册和注销客户端的通道
	register   chan *Client
	unregister chan *Client
}

// NewHub 创建一个新的 Hub 实例
func NewHub() *Hub {
	return &Hub{
		clients:     make(map[string]map[*Client]bool),
		userClients: make(map[uint]map[*Client]bool), // ← 新增
		register:    make(chan *Client),
		unregister:  make(chan *Client),
	}
}

// Run 启动 Hub 的主事件循环
// 这个方法必须在独立的 goroutine 中运行
//
// 设计思想：所有对 clients map 的写操作都通过 channel 汇集到这个单 goroutine 中，
// 避免了并发写 map 导致的 panic（Go 的 map 不是并发安全的）。
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// 将客户端加入所有交易对的订阅列表（用于行情推送）
			for _, clients := range h.clients {
				clients[client] = true
			}
			// 加入用户索引（用于成交推送）
			if client.userID != 0 {
				if h.userClients[client.userID] == nil {
					h.userClients[client.userID] = make(map[*Client]bool)
				}
				h.userClients[client.userID][client] = true
			}
			h.mu.Unlock()
			log.Printf("client registered: connID=%s, userID=%d", client.connID, client.userID)

		case client := <-h.unregister:
			h.mu.Lock()
			// 从交易对订阅中移除
			for _, clients := range h.clients {
				delete(clients, client)
			}
			// 从用户索引中移除
			if client.userID != 0 {
				if userClients, ok := h.userClients[client.userID]; ok {
					delete(userClients, client)
					if len(userClients) == 0 {
						delete(h.userClients, client.userID)
					}
				}
			}
			close(client.send)
			h.mu.Unlock()
			log.Printf("client unregistered: connID=%s, userID=%d", client.connID, client.userID)
		}
	}
}

// BroadcastTicker 向所有订阅了该交易对的客户端推送行情
// 由 Simulator 在每个 tick 时调用
func (h *Hub) BroadcastTicker(ticker *Ticker) {
	data, err := json.Marshal(ticker)
	if err != nil {
		log.Printf("failed to marshal ticker: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	// 只推送给订阅了该交易对的客户端
	if clients, ok := h.clients[ticker.Symbol]; ok {
		for client := range clients {
			select {
			case client.send <- data: // 非阻塞发送
			default:

				// 客户端 send 缓冲区满了，跳过这条消息
				// 避免一个慢客户端拖慢所有推送
			}
		}
	}
}

// SendToUser 向指定用户的所有连接推送消息
// 用于成交回报、余额变动等用户级别的通知
func (h *Hub) SendToUser(userID uint, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	userClients, ok := h.userClients[userID]
	if !ok {
		return // 该用户没有 WebSocket 连接
	}

	for client := range userClients {
		select {
		case client.send <- data:
		default:
			// 跳过慢客户端
		}
	}
}

// clientCount 客户端订阅某个交易对
func (h *Hub) Subscribe(client *Client, symbol string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 如果该交易对还没有订阅集合，先创建一个
	if h.clients[symbol] == nil {
		h.clients[symbol] = make(map[*Client]bool)
	}
	h.clients[symbol][client] = true
	log.Printf("client subscribed to %s", symbol)
}

// Unsubscribe 客户端取消订阅某个交易对
func (h *Hub) Unsubscribe(client *Client, symbol string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[symbol]; ok {
		delete(clients, client)
		log.Printf("client unsubscribed from %s", symbol)
	}
}

// clientCount 返回当前订阅某个交易对的客户端数量
func (h *Hub) clientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 用第一个交易对的订阅数来估算
	for _, clients := range h.clients {
		return len(clients)
	}
	return 0
}
