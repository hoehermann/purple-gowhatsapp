package atom

import (
	"encoding/binary"
)

// MvhdBox - Movie Header Box
// Box Type: mvhd
// Container: Movie Box (moov)
// Mandatory: Yes
// Quantity: Exactly one.
//
// This box defines overall information which is media-independent,
// and relevant to the entire presentationconsidered as a whole.
type MvhdBox struct {
	*Box
	Flags            uint32
	Version          uint8
	CreationTime     uint32
	ModificationTime uint32
	Timescale        uint32
	Duration         uint32
	Rate             Fixed32
	Volume           Fixed16
}

func (b *MvhdBox) parse() error {
	data := b.ReadBoxData()
	b.Version = data[0]
	b.Timescale = binary.BigEndian.Uint32(data[12:16])
	b.Duration = binary.BigEndian.Uint32(data[16:20])
	b.Rate = fixed32(data[20:24])
	b.Volume = fixed16(data[24:26])
	return nil
}
