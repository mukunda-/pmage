package pmage

import (
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

type Converter interface {
	Convert(inputPath string, outputPath string, exportType string) error
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

func (c *converter) Convert(inputPath string, outputPath string, exportType string) (rerr error) {
	yamlPath := changeExt(inputPath, ".yaml")
	var pmageFile PmageFile
	if err := pmageFile.LoadYamlFile(c.Profile, yamlPath); err != nil {
		return err
	}

	product := CreateProduct(c.Profile, &pmageFile)
	inputImage, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer inputImage.Close()
	img, _, err := image.Decode(inputImage)
	if err != nil {
		return err
	}
	if err := product.LoadImage(img); err != nil {
		return err
	}

	var exporter Exporter
	switch exportType {
	case "ca65":
		exporter = &Ca65Exporter{}
	default:
		return fmt.Errorf("Unknown export type \"%s\". Valid export types are [ca65]", exportType)
	}

	if err := exporter.Export(product, outputPath); err != nil {
		return err
	}

	return nil
}
