package atom

import (
	"encoding/binary"
	"fmt"
	"math"
)

// GetDurationString Helper function to print a duration value in the form H:MM:SS.MS
func GetDurationString(duration uint32, timescale uint32) string {
	durationSec := float64(duration) / float64(timescale)

	hours := math.Floor(durationSec / 3600)
	durationSec -= hours * 3600

	minutes := math.Floor(durationSec / 60)
	durationSec -= minutes * 60

	msec := durationSec * 1000
	durationSec = math.Floor(durationSec)
	msec -= durationSec * 1000
	msec = math.Floor(msec)

	str := fmt.Sprintf("%02.0f:%02.0f:%02.0f:%.0f", hours, minutes, durationSec, msec)
	return str
}

// Fixed16 is an 8.8 Fixed Point Decimal notation
type Fixed16 uint16

func (f Fixed16) String() string {
	return fmt.Sprintf("%v", uint16(f)>>8)
}

func fixed16(bytes []byte) Fixed16 {
	return Fixed16(binary.BigEndian.Uint16(bytes))
}

// Fixed32 is a 16.16 Fixed Point Decimal notation
type Fixed32 uint32

func fixed32(bytes []byte) Fixed32 {
	return Fixed32(binary.BigEndian.Uint32(bytes))
}
