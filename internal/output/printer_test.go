package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jackchuka/ccli/internal/output"
)

func TestPrinterJSON(t *testing.T) {
	var buf bytes.Buffer
	p := output.NewPrinter(&buf, output.FormatJSON, true)

	data := map[string]string{"name": "test"}
	if err := p.Print(data); err != nil {
		t.Fatalf("Print: %v", err)
	}

	var got map[string]string
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["name"] != "test" {
		t.Errorf("got name %q, want %q", got["name"], "test")
	}
}

func TestPrinterYAML(t *testing.T) {
	var buf bytes.Buffer
	p := output.NewPrinter(&buf, output.FormatYAML, true)

	data := map[string]string{"name": "test"}
	if err := p.Print(data); err != nil {
		t.Fatalf("Print: %v", err)
	}

	if !strings.Contains(buf.String(), "name: test") {
		t.Errorf("expected YAML output, got:\n%s", buf.String())
	}
}

func TestPrinterText(t *testing.T) {
	var buf bytes.Buffer
	p := output.NewPrinter(&buf, output.FormatText, true)

	if err := p.PrintText("hello world"); err != nil {
		t.Fatalf("PrintText: %v", err)
	}
	if !strings.Contains(buf.String(), "hello world") {
		t.Errorf("expected text output, got: %q", buf.String())
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input string
		want  output.Format
		err   bool
	}{
		{"text", output.FormatText, false},
		{"json", output.FormatJSON, false},
		{"yaml", output.FormatYAML, false},
		{"xml", "", true},
	}
	for _, tt := range tests {
		got, err := output.ParseFormat(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("ParseFormat(%q): error = %v, want error = %v", tt.input, err, tt.err)
		}
		if got != tt.want {
			t.Errorf("ParseFormat(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
