package pmage

import (
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
)

// An Exporter is responsible for writing a product to a specific file format.
// It follows the directives of the product's "exports" field to determine what should be
// written to the output file.
type Exporter interface {
	Export(product *Product, path string) error
}

type Ca65Exporter struct {
	// If this is not set, it will default to the pmf's segment
	// If the pmf's segment is not set, it will default to the profile's default segment.
	Segment string
}

func (e *Ca65Exporter) formatLabel(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Base(name)
	if ext := path.Ext(name); ext != "" {
		name = name[:len(name)-len(ext)]
	}

	var ca65InvalidLabelChars = regexp.MustCompile(`[^a-zA-Z0-9_]`)
	var startsWithDigit = regexp.MustCompile(`^[0-9]`)

	name = ca65InvalidLabelChars.ReplaceAllString(name, "_")
	if startsWithDigit.MatchString(name) {
		name = "P" + name
	}
	return name
}

func (e *Ca65Exporter) outputBytes(w io.Writer, data []byte) error {

	for i := 0; i < len(data); i += 128 {
		sliceEnd := i + 128
		if sliceEnd > len(data) {
			sliceEnd = len(data)
		}
		slice := data[i:sliceEnd]

		content := "\t.byte "
		for j, b := range slice {
			if j > 0 {
				content += ","
			}
			content += fmt.Sprintf("$%02x", b)
		}
		_, err := fmt.Fprintf(w, "%s\n", content)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Ca65Exporter) Export(product *Product, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	defer f.Close()

	segment := e.Segment
	if segment == "" {
		segment = product.Pmf.Segment
		if segment == "" {
			segment = product.Profile.DefaultSegment()
		}
	}

	_, err = fmt.Fprintf(f, "; EXPORTED WITH PMAGE\n"+
		"\t.segment \"%s\"\n"+
		"\n", segment)
	if err != nil {
		return err
	}

	labelBase := e.formatLabel(product.Pmf.Name)

	if product.Pmf.Create&CreateMaskPixels != 0 && len(product.Pixels) > 0 {
		label := fmt.Sprintf("%s_pixels", labelBase)
		data := product.PixelBytes()
		if _, err = fmt.Fprintf(f, "\t.global %s\n%s:\n", label, label); err != nil {
			return err
		}
		if err = e.outputBytes(f, data); err != nil {
			return err
		}
	}

	if product.Pmf.Create&CreateMaskPalette != 0 && len(product.Palette) > 0 {
		label := fmt.Sprintf("%s_palette", labelBase)
		data := product.PaletteBytes()
		if _, err := fmt.Fprintf(f, "\t.global %s\n%s:\n", label, label); err != nil {
			return err
		}

		if err := e.outputBytes(f, data); err != nil {
			return err
		}
	}

	return nil
}
