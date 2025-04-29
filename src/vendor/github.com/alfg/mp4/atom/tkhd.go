package atom

import (
	"encoding/binary"
)

// TkhdBox - Track Header Box
// Box Type: tkhd
// Container: Track Box (trak)
// Mandatory: Yes
// Quantity: Exactly one.
type TkhdBox struct {
	*Box
	Version          byte
	Flags            uint32
	CreationTime     uint32
	ModificationTime uint32
	TrackID          uint32
	Duration         uint32
	Layer            uint16
	AlternateGroup   uint16
	Volume           Fixed16
	Matrix           []byte
	Width, Height    Fixed32
}

func (b *TkhdBox) parse() error {
	data := b.ReadBoxData()
	b.Version = data[0]
	// b.Flags = [3]byte{data[1], data[2], data[3]}
	b.Flags = binary.BigEndian.Uint32(data[0:4])
	b.CreationTime = binary.BigEndian.Uint32(data[4:8])
	b.ModificationTime = binary.BigEndian.Uint32(data[8:12])
	b.TrackID = binary.BigEndian.Uint32(data[12:16])
	b.Duration = binary.BigEndian.Uint32(data[20:24])
	b.Layer = binary.BigEndian.Uint16(data[32:34])
	b.AlternateGroup = binary.BigEndian.Uint16(data[34:36])
	b.Volume = fixed16(data[36:38])
	b.Matrix = data[40:76]
	b.Width = fixed32(data[76:80])
	b.Height = fixed32(data[80:84])
	return nil
}

// GetWidth returns a calculated tkhd width.
func (b *TkhdBox) GetWidth() Fixed32 {
	return b.Width / (1 << 16)
}

// GetHeight returns a calculated tkhd height.
func (b *TkhdBox) GetHeight() Fixed32 {
	return b.Height / (1 << 16)
}
