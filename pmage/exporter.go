package pmage

import "os"

// An Exporter is responsible for writing a product to a specific file format.
// It follows the directives of the product's "exports" field to determine what should be
// written to the output file.
type Exporter interface {
	Export(product *Product, path string) error
}

type Ca65Exporter struct {
	Segment string
}

func (e *Ca65Exporter) Export(product *Product, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	defer f.Close()

	content := "; EXPORTED WITH PMAGE\n"
	content += "\t.segment \"" + e.Segment + "\"\n"
	content += "\n"

	_, err = f.WriteString(content)
	if err != nil {
		return err
	}

	// for i := 0; i < product.Pixels; i++ {

	// }

	return nil
}
