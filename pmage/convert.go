package pmage

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

type Converter interface {
	Convert(inputPath string, outputPath string) error
}

type converter struct {
	Profile *Profile
}

func NewConverter(profile *Profile) Converter {
	return &converter{
		Profile: profile,
	}
}

func changeExt(inputPath string, newExt string) string {
	dir := filepath.Dir(inputPath)
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return filepath.Join(dir, name+newExt)
}

func (c *converter) Convert(inputPath string, outputPath string) (rerr error) {
	yamlPath := changeExt(inputPath, ".yaml")
	var pmageFile PmageFile
	if err := pmageFile.LoadYamlFile(c.Profile, yamlPath); err != nil {
		return err
	}

	file, err := os.Open(inputPath)
	if err != nil {
		return err
	}

	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	err = png.Encode(out, img)
	if err != nil {
		return err
	}

	return nil
}
