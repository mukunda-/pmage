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

func TestMap16(t *testing.T) {

	pmage := `
tiles: 8x8
grouptiles: 16x16
bpp: 4
`

	profile := Profile{"snes"}
	pmf, err := CreatePmageFileFromYamlString(&profile, pmage)
	assert.NoError(t, err)

	p := CreateProduct(&profile, pmf)
	err = p.LoadImage(loadPng("test/flippy16.png"))
	assert.NoError(t, err)

	assert.Equal(t, 1, p.NumTiles())

}
