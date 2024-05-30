package ws

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	pongWait = 10 * time.Second

	pingInterval = (pongWait * 9) / 10
)

type Client struct {
	sync.RWMutex

	id string

	connection *websocket.Conn
	manager    *Manager

	debugEgress chan []byte
	egress      chan []byte
}

func NewClient(conn *websocket.Conn, manager *Manager) (*Client, error) {
	buf := make([]byte, 32)
	_, err := rand.Read(buf)
	if err != nil {
		return nil, err
	}

	wsID := hex.EncodeToString(buf)

	return &Client{
		id:          wsID,
		connection:  conn,
		manager:     manager,
		egress:      make(chan []byte),
		debugEgress: make(chan []byte),
	}, nil
}

func (c *Client) CompareID(another *Client) bool {
	return c.id == another.id
}

func (c *Client) Start() {
	go c.readMessages()
	go c.writeMessages()
}

func (c *Client) readMessages() {
	defer func() {
		c.manager.removeClient(c)
	}()

	c.connection.SetReadLimit(512)

	c.connection.SetPongHandler(c.pongHandler)

	for {
		messageType, _, err := c.connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				slog.Error("unexpected ws close", "error", err)
			}
			break
		}

		if messageType == websocket.CloseMessage {
			slog.Info("client closed socket")
			break
		}
	}
}

func (c *Client) writeMessages() {
	defer func() {
		c.manager.removeClient(c)
	}()

	ticker := time.NewTicker(pingInterval)

	for {
		select {
		case message, ok := <-c.egress:
			if !ok {
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					slog.Error("connetion closed", "error", err)
				}
				return
			}

			if err := c.connection.WriteMessage(websocket.TextMessage, message); err != nil {
				if errors.Is(err, websocket.ErrCloseSent) {
					return
				}
				slog.Error("failed to send message", "error", err)
			}
		case <-ticker.C:
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte(``)); err != nil {
				if !errors.Is(err, websocket.ErrCloseSent) {
					slog.Error("failed to send ping message", "error", err)
				}
				return
			}
		}
	}
}

func (c *Client) StartDebug() {
	go c.readDebugMessages()
	go c.writeDebugMessages()
}

func (c *Client) readDebugMessages() {
	defer func() {
		c.manager.removeDebugClient(c)
	}()

	c.connection.SetReadLimit(512)

	for {
		messageType, _, err := c.connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("unexpected ws close", "error", err)
			}
			break
		}

		if messageType == websocket.CloseMessage {
			slog.Info("client closed socket")
			break
		}
	}
}

func (c *Client) writeDebugMessages() {
	defer func() {
		c.manager.removeDebugClient(c)
	}()

	for {
		message, ok := <-c.debugEgress
		if !ok {
			if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
				slog.Error("connetion closed", "error", err)
			}
			return
		}

		if err := c.connection.WriteMessage(websocket.TextMessage, message); err != nil {
			slog.Error("failed to send message", "error", err)
		}
	}
}

func (c *Client) pongHandler(pongMsg string) error {
	return c.connection.SetReadDeadline(time.Now().Add(pongWait))
}
