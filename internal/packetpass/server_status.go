package packetpass

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/kKar1503/rewired-server-2024/internal/db"
	"github.com/kKar1503/rewired-server-2024/internal/ws"
)

type ServerStatus struct {
	Devices []DeviceStatus `json:"devices"`
	Rooms   []RoomStatus   `json:"rooms"`
}

type DeviceStatus struct {
	ID     uint16 `json:"id"`
	Status uint8  `json:"status"`
}

type RoomStatus struct {
	Name       string `json:"name"`
	Population uint32 `json:"population"`
}

func ServerStatusPasser(ctx context.Context, bytesEgress chan<- []byte) {
	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-ctx.Done():
			slog.Info("exiting go func egress")
			return
		case <-ticker.C:
			if !ws.HasClients() {
				break
			}
			serverStatus := &ServerStatus{}

			devicePairs := []db.DevicePair{}
			result := db.Get().Where(&db.DevicePair{OwnerID: 1}).Joins("InnerGate").Joins("OuterGate").Find(&devicePairs)
			if result.Error != nil {
				slog.Error("failed to retrieve device pair for user", "error", result.Error, "userID", 1)
				continue
			}

			for _, devicePair := range devicePairs {
				serverStatus.Devices = append(serverStatus.Devices, []DeviceStatus{
					{
						ID:     devicePair.InnerGate.GateID,
						Status: devicePair.InnerGate.Status,
					},
					{
						ID:     devicePair.OuterGate.GateID,
						Status: devicePair.OuterGate.Status,
					},
				}...)
			}

			rooms := []db.Room{}
			result = db.Get().Where(&db.Room{OwnerID: 1}).Joins("RoomPopulation").Find(&rooms)
			if result.Error != nil {
				slog.Error("failed to retrieve rooms for user", "error", result.Error, "userID", 1)
				continue
			}

			for _, room := range rooms {
				serverStatus.Rooms = append(serverStatus.Rooms, RoomStatus{
					Name:       room.Name,
					Population: room.RoomPopulation.Population,
				})
			}

			data, err := json.Marshal(serverStatus)
			if err != nil {
				slog.Error("failed to retrieve rooms for user", "error", err, "userID", 1)
				continue
			}

			bytesEgress <- data
		}
	}
}
