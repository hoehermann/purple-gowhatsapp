package atom

import (
	"encoding/binary"
)

// ElstBox - Edit List Box
// Box Type: elst
// Container: Edit Box (edts)
// Mandatory: No
// Quantity: Zero or one
type ElstBox struct {
	*Box
	Version    uint32 // Version of this box.
	EntryCount uint32 // Integer that gives the number of entries.
	Entries    []elstEntry
}

type elstEntry struct {
	SegmentDuration   uint32 // Duration of this edit segment.
	MediaTime         uint32 // Starting time within the media of this edit segment.
	MediaRate         uint16 // Relative rate at which to play the media corresponding to this segment.
	MediaRateFraction uint16
}

func (b *ElstBox) parse() error {
	data := b.ReadBoxData()
	b.Version = binary.BigEndian.Uint32(data[0:4])
	b.EntryCount = binary.BigEndian.Uint32(data[4:8])
	b.Entries = make([]elstEntry, b.EntryCount)

	for i := 0; i < len(b.Entries); i++ {
		b.Entries[i].SegmentDuration = binary.BigEndian.Uint32(data[(8 + 12*i):(12 + 12*i)])
		b.Entries[i].MediaTime = binary.BigEndian.Uint32(data[(12 + 12*i):(16 + 12*i)])
		b.Entries[i].MediaRate = binary.BigEndian.Uint16(data[(16 + 12*i):(18 + 12*i)])
		b.Entries[i].MediaRateFraction = binary.BigEndian.Uint16(data[(18 + 12*i):(20 + 12*i)])
	}
	return nil
}
