package atom

// MdiaBox - Media Box
// Box Type: mdia
// Container: Track Box (trak)
// Mandatory: Yes
// Quantity: Exactly one.
// The media declaration container contains all the objects that declare information
// about the media data within a track.
type MdiaBox struct {
	*Box
	Hdlr *HdlrBox
	Mdhd *MdhdBox
	Minf *MinfBox
}

func (b *MdiaBox) parse() error {
	boxes := readBoxes(b.Reader, b.Start+BoxHeaderSize, b.Size-BoxHeaderSize)

	for _, box := range boxes {
		switch box.Name {
		case "hdlr":
			b.Hdlr = &HdlrBox{Box: box}
			b.Hdlr.parse()

		case "mdhd":
			b.Mdhd = &MdhdBox{Box: box}
			b.Mdhd.parse()

		case "minf":
			b.Minf = &MinfBox{Box: box}
			b.Minf.parse()
		}
	}
	return nil
}
