package atom

import (
	"encoding/binary"
	"fmt"
)

// MdhdBox - Media Header Box
// Box Type: mdhd
// Container: Media Box (mdia)
// Mandatory: Yes
// Quantity: Any number.
//
// The media header declares overall information that is media-independent,
// and relevant to characteristics of the media in a track.
type MdhdBox struct {
	*Box
	Version          byte
	Flags            uint32
	CreationTime     uint32
	ModificationTime uint32
	Timescale        uint32
	Duration         uint32
	Language         uint16
	LanguageString   string
}

func (b *MdhdBox) parse() error {
	data := b.ReadBoxData()
	b.Version = data[0]
	b.Flags = binary.BigEndian.Uint32(data[0:4])
	b.CreationTime = binary.BigEndian.Uint32(data[4:8])
	b.ModificationTime = binary.BigEndian.Uint32(data[8:12])
	b.Timescale = binary.BigEndian.Uint32(data[12:16])
	b.Duration = binary.BigEndian.Uint32(data[16:20])
	b.Language = binary.BigEndian.Uint16(data[20:22])
	b.LanguageString = getLanguageString(b.Language)
	return nil
}

func getLanguageString(language uint16) string {
	var lang [3]uint16
	lang[0] = (language >> 10) & 0x1F
	lang[1] = (language >> 5) & 0x1F
	lang[2] = (language) & 0x1F
	return fmt.Sprintf("%s%s%s",
		string(rune(lang[0]+0x60)),
		string(rune(lang[1]+0x60)),
		string(rune(lang[2]+0x60)))
}
