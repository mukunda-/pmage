package pmage

import (
	"errors"
	"fmt"
	"image"
	"slices"
)

type Color uint32
type Pixel uint32
type MapFlags uint32
type TileIndex struct {
	Flags MapFlags
	Index uint32
}

type ColorFormat int16

const (
	ColorFormatInherit ColorFormat = 0
	ColorFormat32abgr  ColorFormat = 1 // 0xAABBGGRR
	ColorFormat24bgr   ColorFormat = 2 // 0x--BBGGRR

	// Valid for palettes or pixels
	ColorFormat15bgr ColorFormat = 3 // 0b-bbbbbgggggrrrrr

	// Valid for pixels only
	ColorFormatIndexed8 ColorFormat = 4 // 256 colors, 8bpp paletted
	ColorFormatIndexed4 ColorFormat = 5 // 16 colors, 4bpp
	ColorFormatIndexed2 ColorFormat = 6 // 4 colors, 2bpp
	ColorFormatIndexed1 ColorFormat = 7 // 2 colors, 1bpp
)

const (
	MapFlagHflip MapFlags = 1
	MapFlagVflip MapFlags = 2
	MapFlagPrio  MapFlags = 4
)

// A product is the result of the conversion process.
type Product struct {
	Profile *Profile
	Pmf     *PmageFile

	PaletteFormat ColorFormat
	Palette       []Color

	PixelFormat ColorFormat
	Width       int
	Height      int

	// Pixel data starts out in ColorFormat32abgr format when an image is loaded. It is
	// then converted to the desired format.
	Pixels []Pixel

	Map []TileIndex
}

var ErrInvalidImage = errors.New("invalid image")
var ErrUnsupported = errors.New("unsupported operation")
var ErrConversion = errors.New("conversion error")

func CreateProduct(profile *Profile, pmf *PmageFile) *Product {
	return &Product{
		Profile: profile,
		Pmf:     pmf,
	}
}

func (p *Product) LoadImage(img image.Image) error {
	if img.Bounds().Min.X != 0 || img.Bounds().Min.Y != 0 {
		return fmt.Errorf("%w: lower boundary must be zero", ErrInvalidImage)
	}

	p.Width = img.Bounds().Max.X
	p.Height = img.Bounds().Max.Y
	p.Pixels = make([]Pixel, p.Width*p.Height)
	cursor := 0

	for y := 0; y < p.Height; y++ {
		for x := 0; x < p.Width; x++ {
			color := img.At(x, y)
			r, g, b, a := color.RGBA()
			r >>= 8 // Scale to 8-bit
			g >>= 8
			b >>= 8
			a >>= 8
			p.Pixels[cursor] = Pixel((r) | (g << 8) | (b << 16) | (a << 24))
			cursor++
		}
	}
	p.PixelFormat = ColorFormat32abgr

	if err := p.convertPixels(); err != nil {
		return err
	}
	if err := p.tilePixels(); err != nil {
		return err
	}

	if p.Pmf.Bpp <= 8 {
		if err := p.createPalette(); err != nil {
			return err
		}
		if err := p.indexPixels(); err != nil {
			return err
		}
	}

	// if err := p.mapTiles(); err != nil {
	// 	return err
	// }

	return nil
}

func (p *Product) createPalette() error {
	if p.Pmf.Bpp > 8 {
		return fmt.Errorf("%w: bpp too high for palette", ErrConversion)
	}

	type paletteEntry struct {
		index int16
		color Color
	}

	numColors := 0
	maxColors := 1 << p.Pmf.Bpp
	colorMap := make(map[Color]paletteEntry)

	// The palette is initially in 24-bit format. Convert to our profile format.
	convertedPalette := slices.Clone(p.Pmf.Palette)
	convertColors(convertedPalette, p.Profile.GetColorFormat())
	p.PaletteFormat = p.Profile.GetColorFormat()
	numStaticColors := len(convertedPalette)

	// Fixed palette entries
	for _, color := range convertedPalette {
		colorMap[color] = paletteEntry{
			index: int16(numColors),
			color: color,
		}
		numColors++

		if numColors > maxColors {
			return fmt.Errorf("%w: too many colors u	d", ErrConversion)
		}
	}

	for _, pixel := range p.Pixels {
		_, ok := colorMap[Color(pixel)]
		if ok {
			continue
		}

		colorMap[Color(pixel)] = paletteEntry{
			index: int16(numColors),
			color: Color(pixel),
		}
		numColors++

		if numColors > maxColors {
			return fmt.Errorf("%w: too many colors used", ErrConversion)
		}
	}

	p.Palette = make([]Color, maxColors)
	for _, color := range colorMap {
		p.Palette[color.index] = color.color
	}

	slices.SortFunc(p.Palette[numStaticColors:numColors], func(a, b Color) int {
		if a < b {
			return -1
		} else if a > b {
			return 1
		}
		return 0
	})

	return nil
}

func convertColors[T Color | Pixel](colors []T, to ColorFormat) error {
	switch to {
	case ColorFormat15bgr:
		for i, color := range colors {
			r := color & 0xFF
			g := (color >> 8) & 0xFF
			b := (color >> 16) & 0xFF
			colors[i] = T(r>>3 | (g>>3)<<5 | (b>>3)<<10)
		}
	default:
		return fmt.Errorf("%w: unsupported color conversion", ErrConversion)
	}

	return nil
}

// Convert the default 32bgra pixels to the color format of the current profile.
func (p *Product) convertPixels() error {
	if p.PixelFormat != ColorFormat32abgr {
		return fmt.Errorf("%w: convert pixels can only convert from 32abgr", ErrConversion)
	}

	colorFormat := p.Profile.GetColorFormat()
	convertColors(p.Pixels, colorFormat)
	p.PixelFormat = colorFormat

	return nil
}

// Cut the image into tiles specified by the pmage file. The image width will become the
// tile width, the length being a strip of tiles.
func (p *Product) tilePixels() error {
	if p.Pmf.TileWidth <= 1 || p.Pmf.TileHeight <= 1 {
		// Tiling is disabled.
		return nil
	}

	theight := int(p.Pmf.TileHeight)
	twidth := int(p.Pmf.TileWidth)

	if p.Width%twidth != 0 || p.Height%theight != 0 {
		return fmt.Errorf("%w: image size not a multiple of tile size", ErrInvalidImage)
	}

	htiles := p.Width / twidth
	vtiles := p.Height / theight

	newPixels := make([]Pixel, p.Width*p.Height)

	for ty := 0; ty < vtiles; ty++ {
		for tx := 0; tx < htiles; tx++ {
			for py := 0; py < theight; py++ {
				for px := 0; px < twidth; px++ {
					newIndex := ty*twidth*theight*htiles + tx*twidth*theight + py*twidth + px
					oldIndex := (ty*theight+py)*p.Width + tx*twidth + px
					newPixels[newIndex] = p.Pixels[oldIndex]
				}
			}
		}
	}

	p.Pixels = newPixels
	p.Width = twidth
	p.Height = theight * htiles * vtiles

	return nil
}

func (p *Product) indexPixels() error {
	if p.Pmf.Bpp > 8 {
		return fmt.Errorf("%w: bpp too high for indexing", ErrConversion)
	}

	if p.Pmf.Bpp != 2 && p.Pmf.Bpp != 4 && p.Pmf.Bpp != 8 {
		return fmt.Errorf("%w: unsupported bpp for indexing", ErrConversion)
	}

	mapping := make(map[Pixel]int16)
	for i, color := range p.Palette {
		mapping[Pixel(color)] = int16(i)
	}

	// Convert the pixels to palette indexes.
	for i, pixel := range p.Pixels {
		paletteIndex, ok := mapping[pixel]
		if !ok {
			return fmt.Errorf("%w: pixel color not in palette", ErrConversion)
		}
		p.Pixels[i] = Pixel(paletteIndex)
	}

	if p.Pmf.Bpp == 2 {
		p.PixelFormat = ColorFormatIndexed2
	} else if p.Pmf.Bpp == 4 {
		p.PixelFormat = ColorFormatIndexed4
	} else {
		p.PixelFormat = ColorFormatIndexed8
	}

	return nil
}

// Creates a tilemap and eliminates duplicate tiles in the image.
func (p *Product) mapTiles() error {
	if p.Width != int(p.Pmf.TileWidth) {
		return fmt.Errorf("%w: image width must be tile width", ErrConversion)
	}
	if p.Height%int(p.Pmf.TileHeight) != 0 {
		return fmt.Errorf("%w: image height must be divisible by tile height", ErrConversion)
	}

	newPixels := []Pixel{}
	newTileNum := 0
	newMap := []TileIndex{}
	tw, th := int(p.Pmf.TileWidth), int(p.Pmf.TileHeight)

	findTile := func(source []Pixel) (index int, hflip bool, vflip bool) {
		numNewTiles := len(newPixels) / (tw * th)
		for t := 0; t < numNewTiles; t++ {
			if slices.Equal(source, newPixels[t*tw*th:(t+1)*tw*th]) {
				// Matches the tile.
				return t, false, false
			}

			matches := true
			for y := 0; y < th; y++ {
				for x := 0; x < tw; x++ {
					if source[y*tw+x] != newPixels[t*tw*th+y*tw+(tw-x)] {
						matches = false
						break
					}
				}
			}
			if matches {
				return t, true, false
			}

			matches = true
			for y := 0; y < th; y++ {
				for x := 0; x < tw; x++ {
					if source[y*tw+x] != newPixels[t*tw*th+(th-y)*tw+x] {
						matches = false
						break
					}
				}
			}
			if matches {
				return t, false, true
			}

			matches = true
			for y := 0; y < th; y++ {
				for x := 0; x < tw; x++ {
					if source[y*tw+x] != newPixels[t*tw*th+(th-y)*tw+(tw-x)] {
						matches = false
						break
					}
				}
			}
			if matches {
				return t, true, true
			}

		}
		return -1, false, false
	}

	numTiles := p.Height / th
	for t := 0; t < numTiles; t++ {
		pixels := p.Pixels[t*tw*th : (t+1)*tw*th]
		index, hflip, vflip := findTile(pixels)
		if index < 0 {
			newPixels = append(newPixels, pixels...)
			newTileNum++
			newMap = append(newMap, TileIndex{Index: uint32(newTileNum - 1)})
		} else {
			flags := MapFlags(0)
			if hflip {
				flags |= MapFlagHflip
			}
			if vflip {
				flags |= MapFlagVflip
			}
			newMap = append(newMap, TileIndex{Index: uint32(index), Flags: flags})
		}
	}

	p.Pixels = newPixels
	p.Map = newMap

	return nil
}

func (p *Product) NumTiles() int {
	return len(p.Pixels) / int(p.Pmf.TileWidth*p.Pmf.TileHeight)
}

// Convert the pixel data to a byte array.
func (p *Product) PixelBytes() []byte {

	var data []byte

	// Convert to byte array.
	switch p.PixelFormat {
	case ColorFormat15bgr:
		// 2 bytes per pixel
		data = make([]byte, len(p.Pixels)*2)
		for i, pixel := range p.Pixels {
			data[i*2] = byte(pixel)
			data[i*2+1] = byte(pixel >> 8)
		}
	case ColorFormatIndexed8:
		// 1 byte per pixel
		data = make([]byte, len(p.Pixels))
		for i, pixel := range p.Pixels {
			data[i] = byte(pixel)
		}
	case ColorFormatIndexed4:
		// 1 byte per 2 pixels
		data = make([]byte, len(p.Pixels)/2)
		for i := 0; i < len(p.Pixels); i += 2 {
			data[i/2] = byte(p.Pixels[i]) | byte(p.Pixels[i+1]<<4)
		}
	case ColorFormatIndexed2:
		// 1 byte per 4 pixels
		data = make([]byte, len(p.Pixels)/4)
		for i := 0; i < len(p.Pixels); i += 4 {
			data[i/4] = byte(p.Pixels[i]) |
				byte(p.Pixels[i+1]<<2) |
				byte(p.Pixels[i+2]<<4) |
				byte(p.Pixels[i+3]<<6)
		}
	default:
		panic("unimplemented pixel data format")
	}

	return applyCompression(data, p.Pmf.Compression)
}

func (p *Product) PaletteBytes() []byte {

	// Convert to byte array.
	switch p.PaletteFormat {
	case ColorFormat15bgr:
		// 2 bytes per color
		data := make([]byte, len(p.Palette)*2)
		for i, color := range p.Palette {
			data[i*2] = byte(color)
			data[i*2+1] = byte(color >> 8)
		}
		return data
	}

	panic("unimplemented palette data format")
}
