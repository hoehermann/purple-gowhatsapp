package atom

import (
	"encoding/binary"
)

// StsdBox - Sample Description Box
// Box Type: stsd
// Container: Sample Table Box (stbl)
// Mandatory: Yes
// Quantity: Exactly one.
type StsdBox struct {
	*Box
	Version byte
	Flags   uint32
	Avc1    *Avc1Box
}

func (b *StsdBox) parse() error {
	data := b.ReadBoxData()
	b.Version = data[0]
	b.Flags = binary.BigEndian.Uint32(data[0:4])

	boxes := readBoxes(b.Reader, b.Start+BoxHeaderSize+8, b.Size-BoxHeaderSize) // Skip extra 8 bytes.

	for _, box := range boxes {
		switch box.Name {
		case "avc1":
			b.Avc1 = &Avc1Box{Box: box}
			b.Avc1.parse()
		}
	}
	return nil
}
