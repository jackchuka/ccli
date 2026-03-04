package output

import (
	"encoding/json"
	"fmt"
	"io"

	"go.yaml.in/yaml/v3"
)

// Format represents an output format.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
)

// ParseFormat converts a string to a Format.
func ParseFormat(s string) (Format, error) {
	switch s {
	case "text":
		return FormatText, nil
	case "json":
		return FormatJSON, nil
	case "yaml":
		return FormatYAML, nil
	default:
		return "", fmt.Errorf("unknown format: %q (use text, json, or yaml)", s)
	}
}

// Printer writes structured data in the configured format.
type Printer struct {
	out     io.Writer
	format  Format
	noColor bool
}

// NewPrinter creates a new Printer.
func NewPrinter(out io.Writer, format Format, noColor bool) *Printer {
	return &Printer{out: out, format: format, noColor: noColor}
}

// Format returns the printer's format.
func (p *Printer) Format() Format { return p.format }

// NoColor returns whether color is disabled.
func (p *Printer) NoColor() bool { return p.noColor }

// Print writes data in the configured format (JSON or YAML). For text, use PrintText.
func (p *Printer) Print(data any) error {
	switch p.format {
	case FormatJSON:
		enc := json.NewEncoder(p.out)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	case FormatYAML:
		enc := yaml.NewEncoder(p.out)
		return enc.Encode(data)
	default:
		return fmt.Errorf("Print does not support format %q; use PrintText for text", p.format)
	}
}

// PrintText writes a pre-formatted text string.
func (p *Printer) PrintText(s string) error {
	_, err := fmt.Fprintln(p.out, s)
	return err
}
