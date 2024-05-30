package population

import (
	"log/slog"

	"github.com/kKar1503/rewired-server-2024/internal/db"
)

func IncrementPopulation(gateID uint16) {
	device := &db.Device{}
	result := db.Get().Where(&db.Device{GateID: gateID}).First(device)
	if result.Error != nil {
		slog.Error("failed to find the device", "error", result.Error, "gateID", gateID)
		return
	}

	devicePair := &db.DevicePair{}
	result = db.Get().
		Where(&db.DevicePair{InnerGateID: device.ID}).
		Or(&db.DevicePair{OuterGateID: device.ID}).
		Joins("InnerRoom.RoomPopulation").
		First(devicePair)
	if result.Error != nil {
		slog.Error("failed to find the device pair", "error", result.Error, "device.ID", device.ID)
		return
	}

	// Increment only increment for the room that is inside, thus we will only need to increment the inner room population
	devicePair.InnerRoom.RoomPopulation.Population += 1
	result = db.Get().Save(&devicePair.InnerRoom.RoomPopulation)
	if result.Error != nil {
		slog.Error("failed to increment the room population", "error", result.Error, "room.ID", devicePair.InnerRoomID)
		return
	}
}

func DecrementPopulation(gateID uint16) {
	device := &db.Device{}
	result := db.Get().Where(&db.Device{GateID: gateID}).First(device)
	if result.Error != nil {
		slog.Error("failed to find the device", "error", result.Error, "gateID", gateID)
		return
	}

	devicePair := &db.DevicePair{}
	result = db.Get().
		Where(&db.DevicePair{InnerGateID: device.ID}).
		Or(&db.DevicePair{OuterGateID: device.ID}).
		Joins("InnerRoom.RoomPopulation").
		First(devicePair)
	if result.Error != nil {
		slog.Error("failed to find the device pair", "error", result.Error, "device.ID", device.ID)
		return
	}

	// Decrement only decrement for the room that is inside, thus we will only need to decrement the inner room population
	if devicePair.InnerRoom.RoomPopulation.Population == 0 {
		// if 0, no-op
		return
	}

	devicePair.InnerRoom.RoomPopulation.Population -= 1
	result = db.Get().Save(&devicePair.InnerRoom.RoomPopulation)
	if result.Error != nil {
		slog.Error("failed to decrement the room population", "error", result.Error, "room.ID", devicePair.InnerRoomID)
		return
	}
}
