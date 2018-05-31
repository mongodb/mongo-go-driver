package bson

func (r Reader) ValidateNoAllocs() error {
	givenLength := readi32(r[0:4])
	if len(r) < int(givenLength) || givenLength < 0 {
		return ErrInvalidLength
	}
	var pos uint32 = 4
	var val Value
	end := uint32(givenLength)
	for {
		if pos >= end {
			// We've gone off the end of the buffer and we're missing
			// a null terminator.
			return ErrInvalidReadOnlyDocument
		}
		if r[pos] == '\x00' {
			break
		}
		val.start = pos
		pos++
		n, err := r.validateKey(pos, end)
		pos += n
		if err != nil {
			return err
		}
		val.offset = pos
		val.data = r
		n, err = val.validate(true)
		pos += n
		if err != nil {
			return err
		}
	}
	return nil
}
