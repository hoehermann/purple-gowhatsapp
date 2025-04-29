package atom

// MinfBox - Media Information Box
// Box Type: minf
// Container: Media Box (mdia)
// Mandatory: Yes
// Quantity: Exactly one.
//
// This box contains all the objects that declare characteristics information of the
// media in the track.
type MinfBox struct {
	*Box
	Vmhd *VmhdBox
	// Dinf *DinfBox
	Stbl *StblBox
	Hmhd *HmhdBox
}

func (b *MinfBox) parse() error {
	boxes := readBoxes(b.Reader, b.Start+BoxHeaderSize, b.Size-BoxHeaderSize)

	for _, box := range boxes {
		switch box.Name {
		case "vmhd":
			b.Vmhd = &VmhdBox{Box: box}
			b.Vmhd.parse()

		case "stbl":
			b.Stbl = &StblBox{Box: box}
			b.Stbl.parse()

		case "hmhd":
			b.Hmhd = &HmhdBox{Box: box}
			b.Hmhd.parse()
		}
	}
	return nil
}
