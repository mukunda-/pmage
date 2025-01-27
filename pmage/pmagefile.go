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

const (
	CreateMaskNone    CreateMask = 0
	CreateMaskPixels  CreateMask = 1
	CreateMaskMap     CreateMask = 2
	CreateMaskPalette CreateMask = 4
	CreateMaskAll     CreateMask = 0xFFFFFFFF
)

// A pmage file contains conversion options for a single image. The base filename of the
// image matches the base filename of the pmage file.
type PmageFile struct {
	Profile    *Profile
	TileWidth  int16
	TileHeight int16
	Create     CreateMask
	Bpp        int16
	Palette    []Color
}

type pmageFileInput struct {
	Tiles   string `yaml:"tiles"`
	Create  string `yaml:"create"`
	Bpp     int    `yaml:"bpp"`
	Colors  int    `yaml:"colors"`
	Palette string `yaml:"palette"`
}

var ErrInvalidColors = errors.New("bpp is invalid")
var ErrInvalidCreateOption = errors.New("invalid create option")
var ErrInvalidTileSize = errors.New("invalid tile size specified")

func (pf *PmageFile) LoadYamlFile(profile *Profile, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return pf.LoadYaml(profile, file)
}

func (pf *PmageFile) LoadYaml(profile *Profile, reader io.Reader) error {
	pf.Profile = profile
	yf := pmageFileInput{}
	yaml.NewDecoder(reader).Decode(&yf)

	pf.Bpp = int16(yf.Bpp)
	if !profile.IsValidBpp(pf.Bpp) {
		return fmt.Errorf("%w: %d", ErrInvalidColors, pf.Bpp)
	}

	var err error
	pf.TileWidth, pf.TileHeight, err = pf.parseTileSize(yf.Tiles)
	if err != nil {
		return err
	}

	pf.Create, err = pf.parseCreateMask(yf.Create)
	if err != nil {
		return err
	}

	pf.Palette, err = pf.parsePalette(yf.Palette)
	if err != nil {
		return err
	}

	return nil
}

func (pf *PmageFile) parseBpp(input pmageFileInput) (int16, error) {

	return 0, nil
}

var tileFormatX = regexp.MustCompile(`^(\d+)$`)
var tileFormatXbyX = regexp.MustCompile(`^(\d+)x(\d+)$`)

func (pf *PmageFile) parseTileSize(tiles string) (int16, int16, error) {
	if tiles == "" {
		// Default tiles option
		tiles = "8x8"
	}

	if tileFormatX.MatchString(tiles) {
		m := tileFormatX.FindStringSubmatch(tiles)
		w, _ := strconv.Atoi(m[1])
		w = max(w, 1)
		return int16(w), int16(w), nil
	}

	if tileFormatXbyX.MatchString(tiles) {
		m := tileFormatXbyX.FindStringSubmatch(tiles)
		w, _ := strconv.Atoi(m[1])
		h, _ := strconv.Atoi(m[2])
		w = max(w, 1)
		h = max(h, 1)
		return int16(w), int16(h), nil
	}

	return 0, 0, ErrInvalidTileSize
}

func (pf *PmageFile) parseCreateMask(create string) (CreateMask, error) {
	if create == "" {
		// Default create option
		create = "all"
	}

	parts := strings.Split(create, " ")
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
			return CreateMaskNone, fmt.Errorf("%w: %s", ErrInvalidCreateOption, part)
		}
	}

	return mask, nil
}

var matchColor = regexp.MustCompile(`^#?([0-9a-fA-F]{6})$`)

func (pf *PmageFile) parseColor(color string) (Color, error) {
	if matchColor.MatchString(color) {
		matchColor.FindAllStringSubmatch()
		m := matchColor.FindStringSubmatch(color)
		color = m[1]
		r, _ := strconv.ParseUint(color[0:2], 16, 8)
		g, _ := strconv.ParseUint(color[2:4], 16, 8)
		b, _ := strconv.ParseUint(color[4:6], 16, 8)
		return Color((r << 16) | (g << 8) | b), nil
	}
	return Color(0), fmt.Errorf("invalid color: %s", color)
}

func (pf *PmageFile) parsePalette(paletteString string) ([]Color, error) {
	paletteString = strings.TrimSpace(paletteString)
	if paletteString == "" {
		return nil, nil
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
			return nil, err
		}
		palette = append(palette, color)
	}

	return palette, nil
}
