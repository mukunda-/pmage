package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPmageCli(t *testing.T) {
	os.Remove(".testfile-gfx_ifont.asm")

	pmageCli([]string{"pmage/test/gfx_ifont.png", ".testfile-gfx_ifont.asm"})

	contents, err := os.ReadFile(".testfile-gfx_ifont.asm")
	assert.NoError(t, err)

	assert.Contains(t, string(contents), "$d7,$01,$c0,$5d,$ff,$7f,$00,$00")
	assert.Contains(t, string(contents), ".global gfx_ifont_palette")
	assert.Contains(t, string(contents), "gfx_ifont_palette:")
	assert.Contains(t, string(contents), ".global gfx_ifont_pixels")
	assert.Contains(t, string(contents), "gfx_ifont_pixels:")
	assert.Contains(t, string(contents), ".segment \"GRAPHICS\"")

}
