package utils

import (
	"errors"
	"time"
)

func TimeToInt32(t time.Time) (int32, error) {
	unixTimestamp := t.Unix()

	// Ensure the Unix timestamp fits into an int32
	if unixTimestamp <= int64(^uint32(0)>>1) && unixTimestamp >= int64(-1<<(32-1)) {
		return int32(unixTimestamp), nil
	} else {
		return 0, errors.New("time is not within the range of int32")
	}
}
