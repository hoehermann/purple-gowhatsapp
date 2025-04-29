package atom

import (
	"encoding/binary"
	"io"
	"os"
)

const (
	// BoxHeaderSize Size of box header.
	BoxHeaderSize = 8
	UuidSize      = 16
)

// Mp4Reader defines an mp4 reader structure.
type Mp4Reader struct {
	Reader io.ReaderAt
	Ftyp   *FtypBox
	Moov   *MoovBox
	Mdat   *MdatBox
	Uuids  []*UuidBox
	Size   int64

	IsFragmented bool
}

// Parse reads an MP4 reader for atom boxes.
func (m *Mp4Reader) Parse() error {
	if m.Size == 0 {
		if ofile, ok := m.Reader.(*os.File); ok {
			info, err := ofile.Stat()
			if err != nil {
				return err
			}
			m.Size = info.Size()
		}
	}

	boxes := readBoxes(m, int64(0), m.Size)
	for _, box := range boxes {
		switch box.Name {
		case "ftyp":
			m.Ftyp = &FtypBox{Box: box}
			_ = m.Ftyp.parse()
		case "mdat":
			m.Mdat = &MdatBox{Box: box}
		case "moov":
			m.Moov = &MoovBox{Box: box}
			_ = m.Moov.parse()
			m.IsFragmented = m.Moov.IsFragmented
		case "uuid":
			uuidBox := &UuidBox{Box: box}
			_ = uuidBox.parse()
			m.Uuids = append(m.Uuids, uuidBox)
		}
	}
	return nil
}

// ReadBoxAt reads a box from an offset.
func (m *Mp4Reader) ReadBoxAt(offset int64) (boxSize uint32, boxType string) {
	buf := m.ReadBytesAt(BoxHeaderSize, offset)
	if len(buf) < BoxHeaderSize {
		return 0, ""
	}
	boxSize = binary.BigEndian.Uint32(buf[0:4])
	// check malformed data
	if offset+int64(boxSize) > m.Size {
		return 0, ""
	}
	boxType = string(buf[4:8])
	return
}

// ReadBytesAt reads a box at n and offset.
func (m *Mp4Reader) ReadBytesAt(n int64, offset int64) (word []byte) {
	buf := make([]byte, n)
	if _, err := m.Reader.ReadAt(buf, offset); err != nil {
		return
	}
	return buf
}

// Box defines an Atom Box structure.
type Box struct {
	Name        string
	Size, Start int64
	Reader      *Mp4Reader
}

// ReadBoxData reads the box data from an atom box.
func (b *Box) ReadBoxData() []byte {
	if b.Size <= BoxHeaderSize {
		return nil
	}
	return b.Reader.ReadBytesAt(b.Size-BoxHeaderSize, b.Start+BoxHeaderSize)
}

func readBoxes(m *Mp4Reader, start int64, n int64) (l []*Box) {
	for offset := start; offset < start+n; {
		size, name := m.ReadBoxAt(offset)
		if size == 0 || len(name) == 0 {
			break
		}
		b := &Box{
			Name:   name,
			Size:   int64(size),
			Reader: m,
			Start:  offset,
		}
		l = append(l, b)
		offset += int64(size)
	}
	return l
}
