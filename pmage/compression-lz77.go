package pmage

// This compressor implements the LZ77 compression algorithm as expected by the Gameboy
// Advance BIOS functions. See GBATEK LZ77UnCompReadNormalWrite8bit.
type Lz77Compressor struct{}

func (c *Lz77Compressor) Compress(data []byte) []byte {
	result := []byte{}

	var header uint32
	header = 0x10 | (uint32(len(data)) << 8)
	result = append(result, byte(header), byte(header>>8), byte(header>>16), byte(header>>24))

	cursor := 0
	for cursor < len(data) {
		blockflags := 0
		blockbuffer := []byte{}
		for block := 0; block < 8; block++ {
			bestdisp := -1
			bestlen := 0
			for disp := 0; disp < 4096; disp++ {
				if cursor-disp-1 < 0 {
					break
				}

				length := 0
				for length = 0; length < 15+3; length++ {
					if cursor+length >= len(data) {
						break
					}

					if data[cursor+length] != data[cursor-disp-1+length] {
						break
					}
				}

				if length > bestlen && length >= 3 {
					bestlen = length
					bestdisp = disp
				}
			}

			if bestlen >= 3 {
				// Block flags is MSB first
				blockflags |= 1 << (7 - block)
				blockbuffer = append(blockbuffer, byte(((bestdisp>>8)&0x0f)|((bestlen-3)<<4)), byte(bestdisp&0xff))
				cursor += bestlen
			} else {
				blockbuffer = append(blockbuffer, data[cursor])
				cursor++
			}

			if cursor >= len(data) {
				break
			}
		}

		result = append(result, byte(blockflags))
		result = append(result, blockbuffer...)
	}

	return result
}
