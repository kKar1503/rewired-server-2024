package doorpass

import (
	"log/slog"
	"sync"
	"time"

	"github.com/kKar1503/rewired-server-2024/internal/db"
	"gorm.io/gorm"
)

const (
	ALLOWANCE_FRAME = 500 * time.Millisecond
	PASS_FRAME      = 1000 * time.Millisecond
)

type DoorState struct {
	sync.Mutex
	DevicePairID   uint
	InnerGateState *GateState
	OuterGateState *GateState
	InnerRoomID    uint
	OuterRoomID    uint
	LastBlocked    LastBlock
}

type LastBlock struct {
	Gate  LastBlockedGate
	Start *time.Time
	End   *time.Time
}

type LastBlockedGate uint8

const (
	LastBlockedGateNone LastBlockedGate = iota
	LastBlockedGateInner
	LastBlockedGateOuter
)

type GateState struct {
	GateID     uint16
	LastActive *time.Time
}

var doorStates map[uint16]*DoorState

func Init() error {
	devicePairs := []db.DevicePair{}
	result := db.Get().Joins("InnerGate").Joins("OuterGate").Find(&devicePairs)
	if result.Error != nil {
		return result.Error
	}

	initDoorStates := make(map[uint16]*DoorState)
	for _, devicePair := range devicePairs {
		doorState := &DoorState{
			DevicePairID: devicePair.ID,
			InnerGateState: &GateState{
				GateID: devicePair.InnerGate.GateID,
			},
			OuterGateState: &GateState{
				GateID: devicePair.OuterGate.GateID,
			},
			LastBlocked: LastBlock{
				Gate: LastBlockedGateNone,
			},
			InnerRoomID: devicePair.InnerRoomID,
			OuterRoomID: devicePair.OuterRoomID,
		}
		initDoorStates[devicePair.InnerGate.GateID] = doorState
		initDoorStates[devicePair.OuterGate.GateID] = doorState
	}

	doorStates = initDoorStates

	return nil
}

func GateActive(gateID uint16) {
	doorState, ok := doorStates[gateID]
	if !ok {
		return
	}

	doorState.Lock()
	defer doorState.Unlock()

	now := time.Now()

	// inner gate logic
	if gateID == doorState.InnerGateState.GateID {
		// If there was no lastActive for inner gate, just give it the lastActive, this only runs at the very start
		if doorState.InnerGateState.LastActive == nil {
			doorState.InnerGateState.LastActive = &now
			return
		}

		// Check if there is more than the allowance frame
		if now.Sub(*doorState.InnerGateState.LastActive) <= ALLOWANCE_FRAME {
			// if less than allowance frame, we'll just update it assuming that it was just signal error from the emitter
			doorState.InnerGateState.LastActive = &now
			return
		}

		// Since it took more than allowance frame, it means that it was blocked by something, such as a person
		// 1. If there is no last block, we place the last blocked
		if doorState.LastBlocked.Gate == LastBlockedGateNone {
			doorState.LastBlocked.Gate = LastBlockedGateInner
			doorState.LastBlocked.Start = doorState.InnerGateState.LastActive
			doorState.LastBlocked.End = &now
			doorState.InnerGateState.LastActive = &now
			return
		}

		// 2. If there is last block and the last block is the same
		if doorState.LastBlocked.Gate == LastBlockedGateInner {
			// We will just update the last block timing, since this mean the object has not yet passed the door
			doorState.LastBlocked.Start = doorState.InnerGateState.LastActive
			doorState.LastBlocked.End = &now
			doorState.InnerGateState.LastActive = &now
			return
		}

		// 3. If there is last block and the last block is different
		// Last block is different, this means that potentially the user has passed the door, this does not straight away
		// mean that the user has passed, because that the last passed could've been very long ago
		// We utilise the PASS_FRAME to determine what is the maximum allowance of leeway for passing the gate.
		if doorState.InnerGateState.LastActive.Sub(*doorState.LastBlocked.Start) > PASS_FRAME ||
			now.Sub(*doorState.LastBlocked.End) > PASS_FRAME {
			// 3a. Pass thhe PASS_FRAME, this means that the object possibly didn't pass from the other gate to this,
			// but rather a new object is now passing from the other end.
			// This case we'll replace the last block with this new block, pending to wait for the other side get passed within
			// the PASS_FRAME.
			doorState.LastBlocked.Gate = LastBlockedGateInner
			doorState.LastBlocked.Start = doorState.InnerGateState.LastActive
			doorState.LastBlocked.End = &now
			doorState.InnerGateState.LastActive = &now
		}

		// 3b. Doesn't pass the PASS_FRAME, this means that the object did pass through the whole door within the
		// PASS_FRAME, this way we can safely assume that they cross this direction, from outer to inner.
		doorState.LastBlocked.Start = nil
		doorState.LastBlocked.End = nil
		doorState.LastBlocked.Gate = LastBlockedGateNone
		doorState.InnerGateState.LastActive = &now

		err := db.Get().Transaction(func(tx *gorm.DB) error {
			innerRoomPopulation := &db.RoomPopulation{}
			if err := tx.Where(&db.RoomPopulation{RoomID: doorState.InnerRoomID}).First(innerRoomPopulation).Error; err != nil {
				return err
			}
			innerRoomPopulation.Population += 1
			if err := tx.Save(innerRoomPopulation).Error; err != nil {
				return err
			}

			outerRoomPopulation := &db.RoomPopulation{}
			if err := tx.Where(&db.RoomPopulation{RoomID: doorState.OuterRoomID}).First(outerRoomPopulation).Error; err != nil {
				return err
			}
			outerRoomPopulation.Population -= 1
			if err := tx.Save(outerRoomPopulation).Error; err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			slog.Error("something went wrong when updating population", "error", err)
		}

		return
	}

	// outer gate logic
	if gateID == doorState.OuterGateState.GateID {
		// If there was no lastActive for outer gate, just give it the lastActive, this only runs at the very start
		if doorState.OuterGateState.LastActive == nil {
			doorState.OuterGateState.LastActive = &now
			return
		}

		// Check if there is more than the allowance frame
		if now.Sub(*doorState.OuterGateState.LastActive) <= ALLOWANCE_FRAME {
			// if less than allowance frame, we'll just update it assuming that it was just signal error from the emitter
			doorState.OuterGateState.LastActive = &now
			return
		}

		// Since it took more than allowance frame, it means that it was blocked by something, such as a person
		// 1. If there is no last block, we place the last blocked
		if doorState.LastBlocked.Gate == LastBlockedGateNone {
			doorState.LastBlocked.Gate = LastBlockedGateOuter
			doorState.LastBlocked.Start = doorState.OuterGateState.LastActive
			doorState.LastBlocked.End = &now
			doorState.OuterGateState.LastActive = &now
			return
		}

		// 2. If there is last block and the last block is the same
		if doorState.LastBlocked.Gate == LastBlockedGateOuter {
			// We will just update the last block timing, since this mean the object has not yet passed the door
			doorState.LastBlocked.Start = doorState.OuterGateState.LastActive
			doorState.LastBlocked.End = &now
			doorState.OuterGateState.LastActive = &now
			return
		}

		// 3. If there is last block and the last block is different
		// Last block is different, this means that potentially the user has passed the door, this does not straight away
		// mean that the user has passed, because that the last passed could've been very long ago
		// We utilise the PASS_FRAME to determine what is the maximum allowance of leeway for passing the gate.
		if doorState.OuterGateState.LastActive.Sub(*doorState.LastBlocked.Start) > PASS_FRAME ||
			now.Sub(*doorState.LastBlocked.End) > PASS_FRAME {
			// 3a. Pass thhe PASS_FRAME, this means that the object possibly didn't pass from the other gate to this,
			// but rather a new object is now passing from the other end.
			// This case we'll replace the last block with this new block, pending to wait for the other side get passed within
			// the PASS_FRAME.
			doorState.LastBlocked.Gate = LastBlockedGateOuter
			doorState.LastBlocked.Start = doorState.OuterGateState.LastActive
			doorState.LastBlocked.End = &now
			doorState.OuterGateState.LastActive = &now
		}

		// 3b. Doesn't pass the PASS_FRAME, this means that the object did pass through the whole door within the
		// PASS_FRAME, this way we can safely assume that they cross this direction, from inner to outer.
		doorState.LastBlocked.Start = nil
		doorState.LastBlocked.End = nil
		doorState.LastBlocked.Gate = LastBlockedGateNone
		doorState.OuterGateState.LastActive = &now

		err := db.Get().Transaction(func(tx *gorm.DB) error {
			outerRoomPopulation := &db.RoomPopulation{}
			if err := tx.Where(&db.RoomPopulation{RoomID: doorState.OuterRoomID}).First(outerRoomPopulation).Error; err != nil {
				return err
			}
			outerRoomPopulation.Population += 1
			if err := tx.Save(outerRoomPopulation).Error; err != nil {
				return err
			}

			innerRoomPopulation := &db.RoomPopulation{}
			if err := tx.Where(&db.RoomPopulation{RoomID: doorState.InnerRoomID}).First(innerRoomPopulation).Error; err != nil {
				return err
			}
			innerRoomPopulation.Population -= 1
			if err := tx.Save(innerRoomPopulation).Error; err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			slog.Error("something went wrong when updating population", "error", err)
		}

		return
	}
}
