package db

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type User struct {
	gorm.Model
	Name        string       `gorm:"unique"`
	DevicePairs []DevicePair `gorm:"foreignKey:OwnerID"`
	Rooms       []Room       `gorm:"foreignKey:OwnerID"`
}

type Device struct {
	gorm.Model
	GateID     uint16 `gorm:"unique"`
	Status     uint8  // 1 is connected; 0 is disconnected
	DeviceLogs []DeviceLog
}

type DevicePair struct {
	gorm.Model
	InnerGateID uint
	InnerGate   Device `gorm:"foreignKey:InnerGateID"`
	OuterGateID uint
	OuterGate   Device `gorm:"foreignKey:OuterGateID"`
	InnerRoomID uint
	InnerRoom   Room `gorm:"foreignKey:InnerRoomID"`
	OuterRoomID uint
	OuterRoom   Room `gorm:"foreignKey:OuterRoomID"`
	OwnerID     uint
}

type Room struct {
	gorm.Model
	Name           string
	OwnerID        uint
	RoomPopulation RoomPopulation
}

type RoomPopulation struct {
	gorm.Model
	Population uint32
	RoomID     uint
}

type DeviceLog struct {
	gorm.Model
	DeviceID    uint
	LogType     uint8      // 1 is heartbeat; 2 is gate status; 3 is increment; 4 is decrement
	Status      *uint8     // 1 is turn on; 2 is unblocked; 3 is blocked; 4 is faulty; nil when log not status
	TriggerTime *time.Time // trigger time of status; nil when log not status
}

var instance *gorm.DB

func Get() *gorm.DB {
	return instance
}

func Init() error {
	newLogger := logger.New(
		log.New(os.Stdout, "\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      false,       // Include params in the SQL log
			Colorful:                  true,        // Disable color
		},
	)

	newDB, err := gorm.Open(sqlite.Open("rewired.db"), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return err
	}

	err = autoMigrate(newDB)
	if err != nil {
		return err
	}

	instance = newDB
	return nil
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&User{}, &Device{}, &DevicePair{}, &Room{}, &RoomPopulation{}, &DeviceLog{})
}
