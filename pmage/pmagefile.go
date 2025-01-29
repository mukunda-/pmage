package pmage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type CreateMask int
type PixelCompression int

const (
	CreateMaskNone    CreateMask = 0
	CreateMaskPixels  CreateMask = 1
	CreateMaskMap     CreateMask = 2
	CreateMaskPalette CreateMask = 4
	CreateMaskAll     CreateMask = 0xFFFFFFFF
)

const (
	PixelCompressionNone PixelCompression = 0
	PixelCompressionLz77 PixelCompression = 1
)

// A pmage file contains conversion options for a single image. The base filename of the
// image matches the base filename of the pmage file.
type PmageFile struct {
	Profile     *Profile
	TileWidth   int16
	TileHeight  int16
	Create      CreateMask
	Bpp         int16
	Palette     []Color
	Compression PixelCompression
	Name        string
	Segment     string
}

type pmageFileInput struct {
	// Used for the symbols in the output. Not used if name is specified.
	Filename string

	Tiles       string `yaml:"tiles"`
	Export      string `yaml:"export"`
	Bpp         int    `yaml:"bpp"`
	Colors      int    `yaml:"colors"` // Alternate way to specify bpp
	Palette     string `yaml:"palette"`
	Transparent string `yaml:"transparent"` // Alias for palette
	Compression string `yaml:"compression"`
	Name        string `yaml:"name"`
	Segment     string `yaml:"segment"`
}

var ErrInvalidColors = errors.New("bpp is invalid")
var ErrInvalidExportOption = errors.New("invalid export option")
var ErrInvalidTileSize = errors.New("invalid tile size specified")

// Convenience function for loading from a YAML string.
func CreatePmageFileFromYamlString(profile *Profile, data string, filename string) (*PmageFile, error) {
	var pf PmageFile
	err := pf.LoadYamlString(profile, data, filename)
	if err != nil {
		return nil, err
	}
	return &pf, nil
}

func (pf *PmageFile) LoadYamlFile(profile *Profile, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return pf.LoadYaml(profile, file, path)
}

func (pf *PmageFile) LoadYamlString(profile *Profile, data string, filename string) error {
	return pf.LoadYaml(profile, strings.NewReader(data), filename)
}

func (pf *PmageFile) LoadYaml(profile *Profile, reader io.Reader, filename string) error {

	pfinput := pmageFileInput{}
	yaml.NewDecoder(reader).Decode(&pfinput)
	pfinput.Filename = filename
	return pf.Load(profile, pfinput)
}

// The different loading functions funnel down into here.
// File/Content -> Parsing -> Load
func (pf *PmageFile) Load(profile *Profile, pfinput pmageFileInput) error {
	pf.Profile = profile

	if err := pf.parseBpp(pfinput); err != nil {
		return err
	}

	if err := pf.parseTileSize(pfinput); err != nil {
		return err
	}

	if err := pf.parseExportMask(pfinput); err != nil {
		return err
	}

	if err := pf.parsePalette(pfinput); err != nil {
		return err
	}

	if err := pf.parseCompression(pfinput); err != nil {
		return err
	}

	if err := pf.parseName(pfinput); err != nil {
		return err
	}

	return nil
}

// The `bpp` field is computed from the `bpp` or `colors` fields in the input.
func (pf *PmageFile) parseBpp(pfinput pmageFileInput) error {
	bpp := pfinput.Bpp
	if bpp == 0 {
		switch pfinput.Colors {
		case 0:
			bpp = int(pf.Profile.DefaultBpp())
		case 2:
			bpp = 1
		case 4:
			bpp = 2
		case 16:
			bpp = 4
		case 256:
			bpp = 8
		default:
			return fmt.Errorf("%w: %d", ErrInvalidColors, pfinput.Colors)
		}
	}

	pf.Bpp = int16(bpp)
	if !pf.Profile.IsValidBpp(pf.Bpp) {
		return fmt.Errorf("%w: %d", ErrInvalidColors, pf.Bpp)
	}

	return nil
}

var tileFormatX = regexp.MustCompile(`^(\d+)$`)
var tileFormatXbyX = regexp.MustCompile(`^(\d+)x(\d+)$`)

// The tile size options are parsed from the `tiles` field.
func (pf *PmageFile) parseTileSize(input pmageFileInput) error {
	tiles := input.Tiles
	if tiles == "" {
		// Default tiles option
		tiles = "8x8"
	}

	if tileFormatX.MatchString(tiles) {
		m := tileFormatX.FindStringSubmatch(tiles)
		w, _ := strconv.Atoi(m[1])
		w = max(w, 1)
		pf.TileWidth, pf.TileHeight = int16(w), int16(w)
		return nil
	}

	if tileFormatXbyX.MatchString(tiles) {
		m := tileFormatXbyX.FindStringSubmatch(tiles)
		w, _ := strconv.Atoi(m[1])
		h, _ := strconv.Atoi(m[2])
		w = max(w, 1)
		h = max(h, 1)
		pf.TileWidth, pf.TileHeight = int16(w), int16(h)
		return nil
	}

	return ErrInvalidTileSize
}

// The export mask controls what data is exported into the final result. The `export`
// field specifies a space-separated list of targets.
func (pf *PmageFile) parseExportMask(input pmageFileInput) error {
	exports := input.Export
	if exports == "" {
		// Default create option
		exports = "all"
	}

	parts := strings.Split(exports, " ")
	mask := CreateMask(0)

	for _, part := range parts {
		part = strings.ToLower(part)
		part = strings.TrimSpace(part)
		switch part {
		case "all":
			mask |= CreateMaskAll
		case "none":
			mask = CreateMaskNone
		case "pixels":
			mask |= CreateMaskPixels
		case "map":
			mask |= CreateMaskMap
		case "palette":
			mask |= CreateMaskPalette
		default:
			return fmt.Errorf("%w: %s", ErrInvalidExportOption, part)
		}
	}

	pf.Create = mask
	return nil
}

var matchColor = regexp.MustCompile(`^#?([0-9a-fA-F]{6})$`)

// Parse a color from a string. The color can be in the format `#RRGGBB` or `RRGGBB`.
// Alpha not supported here.
func (pf *PmageFile) parseColor(color string) (Color, error) {
	if matchColor.MatchString(color) {
		m := matchColor.FindStringSubmatch(color)
		color = m[1]
		r, _ := strconv.ParseUint(color[0:2], 16, 8)
		g, _ := strconv.ParseUint(color[2:4], 16, 8)
		b, _ := strconv.ParseUint(color[4:6], 16, 8)
		return Color((b << 16) | (g << 8) | r), nil
	}
	return Color(0), fmt.Errorf("invalid color: %s", color)
}

// Pmage files can specify a palette directly. This doesn't need to be a full palette,
// usually it would only be one color. The colors given here are fixed at index 0, 1, 2,
// etc... It's common for Nintendo consoles to treat index 0 as transparent, so this
// feature is most useful for setting the transparent color key.
//
// Hence it has an alias "transparent" which is used to set the transparency color.
//
// This feature is also useful for sharing the same palette between images, since the
// color indexes can be otherwise random when they are implied from an image.
func (pf *PmageFile) parsePalette(pfinput pmageFileInput) error {
	paletteString := strings.TrimSpace(pfinput.Palette)
	if paletteString == "" {
		paletteString = strings.TrimSpace(pfinput.Transparent)
	}

	if paletteString == "" {
		return nil
	}

	parts := strings.Split(paletteString, " ")
	palette := []Color{}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		color, err := pf.parseColor(part)
		if err != nil {
			return err
		}
		palette = append(palette, color)
	}

	pf.Palette = palette
	return nil
}

// The compression field controls the compression encoding used for the pixel data.
func (pf *PmageFile) parseCompression(pfinput pmageFileInput) error {
	enc := strings.TrimSpace(pfinput.Compression)
	enc = strings.ToLower(enc)
	switch enc {
	case "lz77":
		pf.Compression = PixelCompressionLz77
	case "", "none":
		pf.Compression = PixelCompressionNone
	default:
		return fmt.Errorf("invalid compression: %s", enc)
	}

	return nil
}

func (pf *PmageFile) parseName(pfinput pmageFileInput) error {
	pf.Name = pfinput.Name
	if pf.Name == "" {
		pf.Name = pfinput.Filename
	}
	return nil
}

func (pf *PmageFile) parseSegment(pfinput pmageFileInput) error {
	pf.Segment = pfinput.Segment
	return nil
}
