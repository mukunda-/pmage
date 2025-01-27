package pmage

import (
	"errors"
	"fmt"
	"image"
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

func (p *Product) LoadImage(img image.Image) error {
	if img.Bounds().Min.X != 0 || img.Bounds().Min.Y != 0 {
		return fmt.Errorf("%w: lower boundary must be zero", ErrInvalidImage)
	}

	p.Pixels = make([]Pixel, img.Bounds().Max.X*img.Bounds().Max.Y)

	for y := 0; y < img.Bounds().Max.Y; y++ {
		for x := 0; x < img.Bounds().Max.X; x++ {
			color := img.At(x, y)
			r, g, b, a := color.RGBA()
			if r > 255 || g > 255 || b > 255 || a > 255 {
				return fmt.Errorf("%w: color out of range", ErrInvalidImage)
			}
			p.Pixels = append(p.Pixels, Pixel(r+(g<<8)+(b<<16)))
		}
	}
	p.PixelFormat = ColorFormat32abgr

	p.convertPixels()
	p.tilePixels()

	if p.Pmf.Bpp <= 8 {
		p.createPalette()
		p.indexPixels()
	}

	return nil
}

func (p *Product) createPalette() error {
	if p.Pmf.Bpp > 8 {
		return fmt.Errorf("%w: bpp too high for palette", ErrInvalidImage)
	}

	type paletteEntry struct {
		index int16
		color Color
	}

	numColors := 0
	maxColors := 1 << p.Pmf.Bpp
	colorMap := make(map[Color]paletteEntry)

	// Fixed palette entries
	for _, color := range p.Pmf.Palette {
		colorMap[color] = paletteEntry{
			index: int16(numColors),
			color: color,
		}
		numColors++

		if numColors > maxColors {
			return fmt.Errorf("%w: too many colors used", ErrInvalidImage)
		}
	}

	for _, pixel := range p.Pixels {
		colorMap[Color(pixel)] = paletteEntry{
			index: int16(numColors),
			color: Color(pixel),
		}
		numColors++

		if numColors > maxColors {
			return fmt.Errorf("%w: too many colors used", ErrInvalidImage)
		}
	}

	p.Palette = make([]Color, maxColors)
	for _, color := range colorMap {
		p.Palette[color.index] = color.color
	}

	return nil
}

func convertColors(colors []Color, to ColorFormat) error {
	switch colorFormat {
	case ColorFormat15bgr:
		for i, color := range colors {
			r := color & 0xFF
			g := (color >> 8) & 0xFF
			b := (color >> 16) & 0xFF
			colors[i] = Color(r>>3 | (g>>3)<<5 | (b>>3)<<10)
		}
	default:
		return fmt.Errorf("%w: unsupported color conversion", ErrUnsupported)
	}
}

// Convert the default 32bgra pixels to the color format of the current profile.
func (p *Product) convertPixels() error {
	if p.PixelFormat != ColorFormat32abgr {
		return fmt.Errorf("%w: convert pixels can only convert from 32abgr", ErrUnsupported)
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

	for ty := 0; ty <= vtiles; ty++ {
		for tx := 0; tx <= htiles; tx++ {
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
	p.Height = twidth * theight * htiles * vtiles

	return nil
}

func (p *Product) indexPixels() error {
	if p.Pmf.Bpp > 8 {
		return fmt.Errorf("%w: bpp too high for indexing", ErrInvalidImage)
	}

	mapping := make(map[Color]int16)
	for _, pixel := range p.Palette {
		mapping[pixel] = 


	// Convert the pixels to palette indexes.
	newPixels := make([]Pixel, len(p.Pixels))
	for i, pixel := range p.Pixels {
		newPixels[i] = Pixel(p.Palette[pixel])
	}

	p.Pixels = newPixels
	p.PixelFormat = ColorFormatIndexed8

	return nil
}