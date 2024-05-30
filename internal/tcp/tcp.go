package tcp

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"

	"github.com/kKar1503/rewired-server-2024/internal/packet"
)

const MAX_PACKET_LENGTH = 64

type TCP struct {
	listener      net.Listener
	packetsEgress chan<- *packet.RawPacket
	quit          chan interface{}
	wg            sync.WaitGroup
}

func NewTCPServer(port uint16, packetsEgress chan<- *packet.RawPacket) (*TCP, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	return &TCP{
		listener:      listener,
		packetsEgress: packetsEgress,
		quit:          make(chan interface{}),
		wg:            sync.WaitGroup{},
	}, nil
}

func (t *TCP) Close() {
	close(t.quit)
	t.listener.Close()
	t.wg.Wait()
}

func (t *TCP) Start() {
	t.wg.Add(1)
	defer t.wg.Done()

	for {
		conn, err := t.listener.Accept()
		if err != nil {
			select {
			case <-t.quit:
				return
			default:
				slog.Error("server error:", "error", err)
			}
		}

		t.wg.Add(1)
		go t.readConnection(conn)
	}
}

func (t *TCP) readConnection(conn net.Conn) {
	defer t.wg.Done()
	defer conn.Close()
	for {
		rawPacket := &packet.RawPacket{}

		err := rawPacket.ReadPackets(conn)
		if err != nil {
			if errors.Is(err, io.EOF) {
				slog.Debug("socket received EOF", "error", err)
			} else {
				slog.Error("server error:", "error", err)
			}
			break
		}

		if t.packetsEgress != nil {
			t.packetsEgress <- rawPacket
		}
	}

	slog.Debug("finished reading from connection")
}
