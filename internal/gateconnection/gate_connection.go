package gateconnection

import (
	"log/slog"
	"sync"
	"time"

	"github.com/kKar1503/rewired-server-2024/internal/db"
)

type DeviceState struct {
	Connected uint8
	ValidTill time.Time
}

type GateConnectionStates struct {
	sync.Mutex
	DeviceStates map[uint16]*DeviceState
}

func Init() error {
	states = GateConnectionStates{
		Mutex:        sync.Mutex{},
		DeviceStates: make(map[uint16]*DeviceState),
	}

	devices := []db.Device{}

	result := db.Get().Find(&devices)
	if result.Error != nil {
		return result.Error
	}

	for _, d := range devices {
		states.DeviceStates[d.GateID] = &DeviceState{
			Connected: d.Status,
			ValidTill: time.Now().Add(60 * time.Second),
		}
	}

	go invalidateConnection()

	return nil
}

var states GateConnectionStates

func KeepConnected(gateID uint16) {
	states.Lock()
	defer states.Unlock()

	deviceState, ok := states.DeviceStates[gateID]

	if !ok {
		device := &db.Device{}
		db.Get().Where(&db.Device{GateID: gateID}).First(&device)

		if device.ID == 0 {
			slog.Error("unknown gateID provided", "gateID", gateID)
			return
		}
	}

	if deviceState.Connected == 1 {
		// it was connected before, update the valid till
		states.DeviceStates[gateID].ValidTill = time.Now().Add(60 * time.Second)
		return
	}

	// it was not connected before
	device := &db.Device{}
	db.Get().Where(&db.Device{GateID: gateID}).First(&device)

	device.Status = 1
	result := db.Get().Save(device)
	if result.Error != nil {
		slog.Error("failed to save device status update", "error", result.Error)
		return
	}
	if result.RowsAffected == 0 {
		slog.Error("failed to save device status update", "result.RowsAffected", result.RowsAffected)
		return
	}

	states.DeviceStates[gateID].ValidTill = time.Now().Add(60 * time.Second)
	states.DeviceStates[gateID].Connected = 1
}

func invalidateConnection() {
	for {
		states.Lock()

		for gateID := range states.DeviceStates {
			if states.DeviceStates[gateID].Connected == 1 && states.DeviceStates[gateID].ValidTill.Before(time.Now()) {

				device := &db.Device{}
				db.Get().Where(&db.Device{GateID: gateID}).First(&device)

				device.Status = 0
				result := db.Get().Save(device)
				if result.Error != nil {
					slog.Error("failed to save device status update", "error", result.Error)
					return
				}
				if result.RowsAffected == 0 {
					slog.Error("failed to save device status update", "result.RowsAffected", result.RowsAffected)
					return
				}

				states.DeviceStates[gateID].Connected = 0
			}
		}

		states.Unlock()

		time.Sleep(5 * time.Second)
	}
}
