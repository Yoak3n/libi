package dispatch

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/coder/websocket"
)

type WSServer struct {
	mu      sync.RWMutex
	conns   []*websocket.Conn
	server  *http.Server
	running bool
}

func NewWSServer() *WSServer {
	return &WSServer{}
}

func (s *WSServer) Start(addr string) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWS)
	s.server = &http.Server{Addr: addr, Handler: mux}
	log.Printf("WebSocket server listening on %s", addr)
	return s.server.ListenAndServe()
}

func (s *WSServer) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	for _, c := range s.conns {
		c.CloseNow()
	}
	s.conns = nil
	if s.server != nil {
		s.server.Close()
		s.server = nil
	}
}

func (s *WSServer) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *WSServer) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("ws accept error: %v", err)
		return
	}
	s.mu.Lock()
	s.conns = append(s.conns, conn)
	s.mu.Unlock()

	// 保持连接存活，读取客户端消息（忽略内容）
	for {
		_, _, err := conn.Read(r.Context())
		if err != nil {
			s.removeConn(conn)
			conn.CloseNow()
			return
		}
	}
}

func (s *WSServer) removeConn(target *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.conns {
		if c == target {
			s.conns = append(s.conns[:i], s.conns[i+1:]...)
			return
		}
	}
}

func (s *WSServer) Broadcast(msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, conn := range s.conns {
		_ = conn.Write(context.Background(), websocket.MessageText, data)
	}
}
