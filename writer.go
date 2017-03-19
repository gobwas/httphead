package httphead

import "bufio"

func WriteOptions(dest *bufio.Writer, options []Option) {
	for i, opt := range options {
		if i > 0 {
			dest.WriteByte(',')
		}

		writeSanitized(dest, opt.Name)

		for key, val := range opt.Parameters {
			dest.WriteByte(';')
			writeSanitized(dest, key)
			if val != "" {
				dest.WriteByte('=')
				writeSanitized(dest, val)
			}
		}
	}
}

func writeSanitized(bw *bufio.Writer, s string) {
	if bts := sanitize(s); bts != nil {
		bw.Write(bts)
	} else {
		bw.WriteString(s)
	}
}

// sanitize prepares quouted string from s with escaped separator characters.
func sanitize(s string) []byte {
	var bts []byte
	var pos int
	for i := 0; i < len(s); i++ {
		c := s[i]
		if OctetTypes[c].IsSeparator() || OctetTypes[c].IsSpace() {
			if bts == nil {
				bts = make([]byte, 1, len(s)+2+1+4)
				bts[0] = '"'
			}
			bts = append(bts, s[pos:i]...)
			bts = append(bts, '\\', s[i])
			pos = i + 1
		}
	}
	if bts != nil {
		bts = append(bts, s[pos:]...)
		bts = append(bts, '"')
	}
	return bts
}
