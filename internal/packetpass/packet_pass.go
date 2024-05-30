package packetpass

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/kKar1503/rewired-server-2024/internal/db"
	"github.com/kKar1503/rewired-server-2024/internal/doorpass/v1"
	"github.com/kKar1503/rewired-server-2024/internal/gateconnection"
	"github.com/kKar1503/rewired-server-2024/internal/packet"
	"github.com/kKar1503/rewired-server-2024/internal/population"
)

func PacketPasser(ctx context.Context, packetIngress <-chan *packet.RawPacket, bytesEgress chan<- []byte) {
	for {
		select {
		case <-ctx.Done():
			slog.Info("exiting go func egress")
			return
		case rawPacket := <-packetIngress:
			switch rawPacket.PacketType {
			case packet.PacketTypeIncrement:
				incrementPacket := &packet.IncrementPacket{}
				err := incrementPacket.Parse(rawPacket)
				if err != nil {
					slog.Error("failed to parse incrementPacket", "error", err)
					break
				}

				slog.Info("received an increment", "gateID", incrementPacket.GateID)
				if jsonData, err := json.Marshal(incrementPacket); err == nil {
					bytesEgress <- jsonData
				}

				device := &db.Device{}
				result := db.Get().Where(&db.Device{GateID: incrementPacket.GateID}).First(device)
				if result.Error != nil {
					slog.Error("failed to find the device", "error", result.Error, "gateID", incrementPacket.GateID)
					break
				}

				deviceLog := &db.DeviceLog{DeviceID: device.ID, LogType: 3}
				result = db.Get().Create(deviceLog)
				if result.Error != nil {
					slog.Error("failed to create the device log",
						"error",
						result.Error,
						"device ID",
						device.ID,
						"gateID",
						incrementPacket.GateID,
					)
					break
				}

				population.IncrementPopulation(incrementPacket.GateID)
			case packet.PacketTypeDecrement:
				decrementPacket := &packet.DecrementPacket{}
				err := decrementPacket.Parse(rawPacket)
				if err != nil {
					slog.Error("failed to parse decrementPacket", "error", err)
					break
				}

				slog.Info("received a decrement", "gateID", decrementPacket.GateID)
				if jsonData, err := json.Marshal(decrementPacket); err == nil {
					bytesEgress <- jsonData
				}

				device := &db.Device{}
				result := db.Get().Where(&db.Device{GateID: decrementPacket.GateID}).First(device)
				if result.Error != nil {
					slog.Error("failed to find the device", "error", result.Error, "gateID", decrementPacket.GateID)
					break
				}

				deviceLog := &db.DeviceLog{DeviceID: device.ID, LogType: 4}
				result = db.Get().Create(deviceLog)
				if result.Error != nil {
					slog.Error("failed to create the device log",
						"error",
						result.Error,
						"device ID",
						device.ID,
						"gateID",
						decrementPacket.GateID,
					)
				}

				population.DecrementPopulation(decrementPacket.GateID)
			case packet.PacketTypeHeartbeat:
				heartbeatPacket := &packet.HeartbeatPacket{}
				err := heartbeatPacket.Parse(rawPacket)
				if err != nil {
					slog.Error("failed to parse heartbeatPacket", "error", err)
					break
				}

				slog.Info("received a heartbeat", "gateID", heartbeatPacket.GateID)
				if jsonData, err := json.Marshal(heartbeatPacket); err == nil {
					bytesEgress <- jsonData
				}

				device := &db.Device{}
				result := db.Get().Where(&db.Device{GateID: heartbeatPacket.GateID}).First(device)
				if result.Error != nil {
					slog.Error("failed to find the device", "error", result.Error, "gateID", heartbeatPacket.GateID)
					break
				}

				deviceLog := &db.DeviceLog{DeviceID: device.ID, LogType: 1}
				result = db.Get().Create(deviceLog)
				if result.Error != nil {
					slog.Error("failed to create the device log",
						"error",
						result.Error,
						"device ID",
						device.ID,
						"gateID",
						heartbeatPacket.GateID,
					)
					break
				}

				gateconnection.KeepConnected(heartbeatPacket.GateID)
			case packet.PacketTypeGateStatus:
				gateStatusPacket := &packet.GateStatusPacket{}
				err := gateStatusPacket.Parse(rawPacket)
				if err != nil {
					slog.Error("failed to parse gateStatusPacket", "error", err)
					break
				}

				slog.Info("received a status",
					"gateID",
					gateStatusPacket.GateID,
					"status",
					gateStatusPacket.Status,
					"timestamp",
					gateStatusPacket.TriggerTime,
				)
				if jsonData, err := json.Marshal(gateStatusPacket); err == nil {
					bytesEgress <- jsonData
				}

				device := &db.Device{}
				result := db.Get().Where(&db.Device{GateID: gateStatusPacket.GateID}).First(device)
				if result.Error != nil {
					slog.Error("failed to find the device", "error", result.Error, "gateID", gateStatusPacket.GateID)
					break
				}

				deviceLog := &db.DeviceLog{
					DeviceID:    device.ID,
					LogType:     2,
					Status:      (*uint8)(&gateStatusPacket.Status),
					TriggerTime: &gateStatusPacket.TriggerTime,
				}
				result = db.Get().Create(deviceLog)
				if result.Error != nil {
					slog.Error("failed to create the device log",
						"error",
						result.Error,
						"device ID",
						device.ID,
						"gateID",
						gateStatusPacket.GateID,
					)
				}

				if gateStatusPacket.Status == packet.GateStatusUnblocked {
					doorpass.GateActive(gateStatusPacket.GateID)
				}
			}
		}
	}
}
