package unique

import (
	"github.com/google/uuid"
	"github.com/sony/sonyflake"
	"time"
)

var flake = sonyflake.NewSonyflake(sonyflake.Settings{
	StartTime: time.Date(2019, 8, 7, 0, 0, 0, 0, time.Local),
})

func ID() uint64 {
	id, err := flake.NextID()
	if err != nil {
		return 0
	}
	return id
}

func Uuid() string {
	return uuid.New().String()
}
