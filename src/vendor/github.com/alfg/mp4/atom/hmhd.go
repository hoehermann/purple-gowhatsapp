package atom

import "encoding/binary"

// HmhdBox - Hint Media Header Box
// Box Type: hmhd
// Container: Media Information Box (minf).
// Mandatory: Yes
// Quantity: Exactly one specific media header shall be present.
//
// Contains general information, independent of the protocol, for hint tracks.
type HmhdBox struct {
	*Box
	Version    byte
	MaxPDUSize uint16
	AvgPDUSize uint16
	MaxBitrate uint32
	AvgBitrate uint32
}

func (b *HmhdBox) parse() error {
	data := b.ReadBoxData()
	b.Version = data[0]
	b.MaxPDUSize = binary.BigEndian.Uint16(data[0:2])
	b.AvgPDUSize = binary.BigEndian.Uint16(data[2:4])
	b.MaxBitrate = binary.BigEndian.Uint32(data[4:8])
	b.AvgBitrate = binary.BigEndian.Uint32(data[8:12])
	return nil
}
