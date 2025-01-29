package pmage

type Compressor interface {
	Compress(data []byte) []byte
}

func applyCompression(data []byte, comp PixelCompression) []byte {
	switch comp {
	case PixelCompressionLz77:
		compressor := Lz77Compressor{}
		return compressor.Compress(data)
	case PixelCompressionNone:
		return data
	default:
		panic("unknown compression scheme")
	}
}
