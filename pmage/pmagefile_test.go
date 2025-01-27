package pmage

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadingPmage1(t *testing.T) {
	profile := &Profile{System: SystemSnes}

	{
		// Valid data.
		var pmageFile1 = `
tiles: 8x16
create: pixels
bpp: 4
`
		var pf PmageFile
		assert.NoError(t, pf.LoadYaml(profile, strings.NewReader(pmageFile1)))
		assert.Equal(t, int16(8), pf.TileWidth)
		assert.Equal(t, int16(16), pf.TileHeight)
		assert.Equal(t, CreateMaskPixels, pf.Create)
		assert.Equal(t, int16(4), pf.Bpp)
	}

	{
		// Bpp must be a power of 2.
		var pmageFile2 = `
tiles: 8x16
create: pixels
bpp: 3
`
		var pf PmageFile
		assert.Error(t, pf.LoadYaml(profile, strings.NewReader(pmageFile2)))
	}

}
