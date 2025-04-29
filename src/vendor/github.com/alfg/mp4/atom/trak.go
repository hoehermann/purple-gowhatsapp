package atom

// TrakBox - Track Box
// Box Type: tkhd
// Container: Movie Box (moov)
// Mandatory: Yes
// Quantity: One or more.
type TrakBox struct {
	*Box
	// SamplesDuration
	// SamplesSize
	// SampleGroupsInfo

	Tkhd *TkhdBox
	Mdia *MdiaBox
	Edts *EdtsBox
}

func (b *TrakBox) parse() error {
	boxes := readBoxes(b.Reader, b.Start+BoxHeaderSize, b.Size-BoxHeaderSize)

	for _, box := range boxes {
		switch box.Name {
		case "tkhd":
			b.Tkhd = &TkhdBox{Box: box}
			b.Tkhd.parse()

		case "mdia":
			b.Mdia = &MdiaBox{Box: box}
			b.Mdia.parse()

		case "edts":
			b.Edts = &EdtsBox{Box: box}
			b.Edts.parse()
		}
	}
	return nil
}
