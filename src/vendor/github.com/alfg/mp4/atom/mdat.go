package atom

// MdatBox - Media Data Box
// Box Type: mdat
// Container: File
// Mandatory: No
// Quantity: Any number.
//
// A container box which can hold the actual media data for a presentation (mdat).
type MdatBox struct {
	*Box
}
