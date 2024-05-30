package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/kKar1503/rewired-server-2024/internal/db"
	"github.com/kKar1503/rewired-server-2024/internal/doorpass/v1"
	"github.com/kKar1503/rewired-server-2024/internal/gateconnection"
	"github.com/kKar1503/rewired-server-2024/internal/packet"
	"github.com/kKar1503/rewired-server-2024/internal/packetpass"
	"github.com/kKar1503/rewired-server-2024/internal/settings"
	"github.com/kKar1503/rewired-server-2024/internal/tcp"
	"github.com/kKar1503/rewired-server-2024/internal/ws"
)

func main() {
	flag.UintVar(&settings.Get().TCPPort, "tcpport", 42069, "port number that the tcp server will serve in")
	flag.UintVar(&settings.Get().WSPort, "wsport", 80, "port number that the tcp server will serve in")
	flag.StringVar(&settings.Get().Origins, "origins", "*", "the origins that is allowed on the server, seprated by commas")
	flag.Parse()

	log.SetOutput(os.Stdout)
	log.SetPrefix("\n")
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	err := db.Init()
	if err != nil {
		slog.Error("failed to create db", "error", err)
		os.Exit(1)
	}

	err = gateconnection.Init()
	if err != nil {
		slog.Error("failed to init gateconnection", "error", err)
		os.Exit(1)
	}

	err = doorpass.Init()
	if err != nil {
		slog.Error("failed to init doorpass", "error", err)
		os.Exit(1)
	}

	wg := &sync.WaitGroup{}
	packetsEgress := make(chan *packet.RawPacket)
	debugWsEgress := make(chan []byte)
	wsEgress := make(chan []byte)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	wg.Add(1)
	go func() {
		defer wg.Done()
		// pack packets from packet.RawPacket into []byte for ws to egress for debugging
		packetpass.PacketPasser(ctx, packetsEgress, debugWsEgress)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		// pack packets from packet.RawPacket into []byte for ws to egress for debugging
		packetpass.ServerStatusPasser(ctx, wsEgress)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		wsServer := ws.NewManager(debugWsEgress, wsEgress)
		server := &http.Server{Addr: fmt.Sprintf(":%d", settings.Get().WSPort)}

		http.HandleFunc("/debug", wsServer.ServeDebugWS)
		http.HandleFunc("/ws", wsServer.ServeWS)

		go func() {
			<-ctx.Done()
			err := server.Shutdown(context.Background())
			if err != nil {
				slog.Error("failed to shutdown http ws server", "error", err)
			}
		}()

		slog.Info("starting ws server", "port", server.Addr)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server closed unexpectedly", "error", err)
			stop()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		server, err := tcp.NewTCPServer(uint16(settings.Get().TCPPort), packetsEgress)
		if err != nil {
			slog.Error("server failed to start", "error", err)
			stop()
		}

		go func() {
			for {
				<-ctx.Done()
				server.Close()
				return
			}
		}()

		server.Start()
		slog.Info("exiting tcp serving")
	}()

	<-ctx.Done()
	wg.Wait()
	time.Sleep(1 * time.Second)
	os.Exit(1)
}
