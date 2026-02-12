package printing

import "bytes"

// TextReceipt: ESC/POS init + text + newline + partial cut.
func TextReceipt(text string) []byte {
	var b bytes.Buffer
	b.Write([]byte{0x1b, 0x40}) // ESC @
	b.WriteString(text)
	if len(text) == 0 || text[len(text)-1] != '\n' {
		b.WriteByte('\n')
	}
	b.Write([]byte{0x1d, 0x56, 0x00}) // GS V 0
	return b.Bytes()
}
