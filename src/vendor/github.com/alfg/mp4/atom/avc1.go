package atom

// Avc1Box defines the avc1 box structure.
type Avc1Box struct {
	*Box
	Version byte
}

func (b *Avc1Box) parse() error {
	data := b.ReadBoxData()
	b.Version = data[0]
	return nil
}
