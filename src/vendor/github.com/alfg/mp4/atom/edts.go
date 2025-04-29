package atom

// EdtsBox - Edit Box
// Box Type: edts
// Container: Track Box (trak)
// Mandatory: No
// Quantity: Zero or one
type EdtsBox struct {
	*Box
	Elst *ElstBox
}

func (b *EdtsBox) parse() error {
	boxes := readBoxes(b.Reader, b.Start+BoxHeaderSize, b.Size-BoxHeaderSize)

	for _, box := range boxes {
		switch box.Name {
		case "elst":
			b.Elst = &ElstBox{Box: box}
			b.Elst.parse()
		}
	}
	return nil
}
