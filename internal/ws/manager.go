package ws

import (
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/kKar1503/rewired-server-2024/internal/settings"
	"github.com/kKar1503/rewired-server-2024/internal/utils"
)

var websocketUpgrader = websocket.Upgrader{
	CheckOrigin:     checkOrigin,
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Manager struct {
	clients      []*Client
	debugClients []*Client
	sync.RWMutex

	debugWsIngress <-chan []byte
	wsIngress      <-chan []byte
}

var connectedClients atomic.Int32

func init() {
	connectedClients.Store(0)
}

func NewManager(debugWsIngress, wsIngress <-chan []byte) *Manager {
	m := &Manager{
		clients:      make([]*Client, 0, 10),
		debugClients: make([]*Client, 0, 10),
	}

	if debugWsIngress != nil {
		m.debugWsIngress = debugWsIngress
		go m.debugWsDataPass()
	}

	if wsIngress != nil {
		m.wsIngress = wsIngress
		go m.wsDataPass()
	}

	return m
}

func HasClients() bool {
	return connectedClients.Load() != 0
}

func (m *Manager) wsDataPass() {
	for {
		data := <-m.wsIngress
		m.RLock()
		for _, c := range m.clients {
			c.egress <- data
		}
		m.RUnlock()
	}
}

func (m *Manager) debugWsDataPass() {
	for {
		data := <-m.debugWsIngress
		m.RLock()
		for _, c := range m.debugClients {
			c.debugEgress <- data
		}
		m.RUnlock()
	}
}

func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("failed to upgrade ws connection", "error", err)
		return
	}

	client, err := NewClient(conn, m)
	if err != nil {
		slog.Error("failed to create a ws client", "error", err)
		conn.Close()
		return
	}

	m.addClient(client)

	client.Start()
}

func (m *Manager) addClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	m.clients = append(m.clients, client)
	connectedClients.Add(1)
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	for i, c := range m.clients {
		if !client.CompareID(c) {
			continue
		}
		client.connection.Close()
		m.clients = utils.RemoveFromSlice(m.clients, i)
		connectedClients.Add(-1)
	}
}

func (m *Manager) ServeDebugWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("failed to upgrade ws connection", "error", err)
		return
	}

	client, err := NewClient(conn, m)
	if err != nil {
		slog.Error("failed to create a ws client", "error", err)
		conn.Close()
		return
	}

	slog.Info("client connected", "id", client.id)

	m.addDebugClient(client)

	client.StartDebug()
}

func (m *Manager) addDebugClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	m.debugClients = append(m.debugClients, client)
}

func (m *Manager) removeDebugClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	for i, c := range m.debugClients {
		if !client.CompareID(c) {
			continue
		}
		slog.Info("client disconnecting", "id", client.id)
		client.connection.Close()
		m.debugClients = utils.RemoveFromSlice(m.debugClients, i)
	}
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")

	origins := settings.Get().Origins

	if origins == "*" {
		return true
	}

	if !strings.Contains(origins, ",") {
		// single origin
		return origins == origin
	}

	return slices.Contains(strings.Split(origins, ","), origin)
}
