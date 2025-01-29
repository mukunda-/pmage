package pmage

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLz77CompressionSimple(t *testing.T) {
	data := []byte{1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2, 3}

	compressor := Lz77Compressor{}

	compressed := compressor.Compress(data)

	// header+flags+3bytes (raw blocks)+2bytes (compressed block)
	assert.Len(t, compressed, 4+1+3+2)
}

func decompressLz77(compressed []byte) []byte {
	readpos := 0
	getbyte := func() byte {
		if readpos >= len(compressed) {
			panic("out of data")
		}
		b := compressed[readpos]
		readpos++
		return b
	}
	result := []byte{}
	if getbyte() != 0x10 {
		panic("invalid header")
	}

	originalLength := int(getbyte()) | (int(getbyte()) << 8) | (int(getbyte()) << 16)

	for len(result) < originalLength {
		blockflags := getbyte()

		for block := 0; block < 8; block++ {
			if len(result) >= originalLength {
				// All data decompressed - terminate the block cycle
				break
			}

			if blockflags&(1<<(7-block)) != 0 {
				// Compressed block
				a := getbyte()
				disp := int(a&0xF) << 8
				copylength := (int(a) >> 4) + 3
				disp |= int(getbyte())
				disp += 1
				if disp >= 33000 {
					panic("invalid displacement")
				}

				resultpos := len(result)
				for i := 0; i < copylength; i++ {

					result = append(result, result[resultpos-disp+i])
				}
			} else {
				// Raw block
				result = append(result, getbyte())
			}
		}
	}

	return result
}

func TestLz77CompressionRandom(t *testing.T) {
	for test := 0; test < 10; test++ {
		original := []byte{}
		for i := 0; i < 6000+test; i++ {
			value := byte(rand.Intn(1+(test%10)) << (test % 4))
			original = append(original, value)
		}
		compressor := Lz77Compressor{}
		compressed := compressor.Compress(original)
		decompressed := decompressLz77(compressed)
		assert.Equal(t, original, decompressed)
	}
}
