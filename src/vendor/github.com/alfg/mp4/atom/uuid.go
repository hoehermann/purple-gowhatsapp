package atom

import (
	"errors"
)

// UuidBox - Uuid Box
// Box Type: uuid
type UuidBox struct {
	*Box
	Uuid []byte
	Data []byte
}

func (b *UuidBox) parse() error {
	data := b.ReadBoxData()
	if len(data) < UuidSize {
		return errors.New("not enough data")
	}
	b.Uuid = data[:UuidSize]
	b.Data = data[UuidSize:]
	return nil
}
