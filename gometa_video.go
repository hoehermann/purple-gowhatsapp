//go:build gometa

package main

// Minimal MP4 box walker to pull width/height/duration out of a video before sending it over the
// Facebook E2EE transport. Messenger clients render an armadillo VideoMessage much better when the
// dimensions and duration are populated (mirrors the thumbnail-w/h requirement in sendImageE2EE).
// We intentionally avoid a full mp4 decode: we only read the moov/mvhd (duration) and the first
// video trak's tkhd (display width/height, stored as 16.16 fixed point).

func be32(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

// findBoxes returns the payloads (header stripped) of every top-level box named `name` in data.
func findBoxes(data []byte, name string) [][]byte {
	var out [][]byte
	i := 0
	for i+8 <= len(data) {
		size := int(be32(data[i:]))
		typ := string(data[i+4 : i+8])
		hdr := 8
		if size == 1 { // 64-bit largesize; assume the high 32 bits are zero (< 4 GiB)
			if i+16 > len(data) {
				break
			}
			size = int(be32(data[i+12:]))
			hdr = 16
		} else if size == 0 { // box extends to end of data
			size = len(data) - i
		}
		if size < hdr || i+size > len(data) {
			break
		}
		if typ == name {
			out = append(out, data[i+hdr:i+size])
		}
		i += size
	}
	return out
}

// videoInfoMP4 extracts (width, height, seconds) from an mp4. Returns zeros for any field it can't
// parse; the caller substitutes defaults so the send still goes through.
func videoInfoMP4(data []byte) (width, height, seconds uint32) {
	moovs := findBoxes(data, "moov")
	if len(moovs) == 0 {
		return
	}
	moov := moovs[0]

	if mvhds := findBoxes(moov, "mvhd"); len(mvhds) > 0 {
		mvhd := mvhds[0]
		if len(mvhd) >= 4 {
			ver := mvhd[0]
			var ts, dur uint32
			if ver == 1 {
				// version 1: creation(8) mod(8) timescale(4) duration(8)
				if len(mvhd) >= 32 {
					ts = be32(mvhd[20:])
					dur = be32(mvhd[28:]) // low 32 bits of the 64-bit duration
				}
			} else {
				// version 0: creation(4) mod(4) timescale(4) duration(4)
				if len(mvhd) >= 20 {
					ts = be32(mvhd[12:])
					dur = be32(mvhd[16:])
				}
			}
			if ts > 0 {
				seconds = dur / ts
			}
		}
	}

	// Take the first trak whose tkhd reports non-zero display dimensions (the video track).
	for _, trak := range findBoxes(moov, "trak") {
		tkhds := findBoxes(trak, "tkhd")
		if len(tkhds) == 0 {
			continue
		}
		tk := tkhds[0]
		// width/height are the final 8 bytes of tkhd, each a 16.16 fixed-point value.
		if len(tk) >= 8 {
			w := be32(tk[len(tk)-8:]) >> 16
			h := be32(tk[len(tk)-4:]) >> 16
			if w > 0 && h > 0 {
				width, height = w, h
				break
			}
		}
	}
	return
}
