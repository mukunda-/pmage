package pmage

import (
	"image"
	"image/png"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func loadPng(filename string) image.Image {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		panic(err)
	}

	return img
}

func TestCreatePalette(t *testing.T) {
	pmage := `
tiles: 8x8
colors: 4
transparent: "ffffff"
compression: none
`
	pmf, err := CreatePmageFileFromYamlString(&Profile{"snes"}, pmage, "test.yaml")
	assert.NoError(t, err)

	p := CreateProduct(&Profile{"snes"}, pmf)
	err = p.LoadImage(loadPng("test/gfx_ifont.png"))
	assert.NoError(t, err)

	assert.Len(t, p.Palette, 4)

	exportedPalette := p.PaletteBytes()
	assert.Equal(t, []byte{0xff, 0x7f, 0xC0, 0x5D, 0x00, 0x00, 0x00, 0x00}, exportedPalette)

}

// Mapping is not implemented, for now.
// func TestMap16(t *testing.T) {

// 	pmage := `
// tiles: 8x8
// grouptiles: 16x16
// bpp: 4
// `

// 	profile := Profile{"snes"}
// 	pmf, err := CreatePmageFileFromYamlString(&profile, pmage)
// 	assert.NoError(t, err)

// 	p := CreateProduct(&profile, pmf)
// 	err = p.LoadImage(loadPng("test/flippy16.png"))
// 	assert.NoError(t, err)

// 	assert.Equal(t, 1, p.NumTiles())

// }
