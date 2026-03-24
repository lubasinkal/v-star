package reader

import (
	"testing"
)

func TestParseFastInt_Valid(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"0", 0},
		{"123", 123},
		{"1", 1},
		{"999999", 999999},
		{"-5", -5},
		{"-100", -100},
		{"-0", 0},
	}
	for _, tt := range tests {
		got, ok := parseFastInt([]byte(tt.input))
		if !ok {
			t.Errorf("parseFastInt(%q) returned false, want true", tt.input)
		}
		if got != tt.want {
			t.Errorf("parseFastInt(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestParseFastInt_Invalid(t *testing.T) {
	tests := []string{
		"",
		"abc",
		"12abc34",
		"-",
		"12.34",
		" 123",
		"123 ",
		"--5",
		"1-2",
	}
	for _, input := range tests {
		_, ok := parseFastInt([]byte(input))
		if ok {
			t.Errorf("parseFastInt(%q) returned true, want false", input)
		}
	}
}

func TestParseFastFloat_Valid(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"0", 0},
		{"123.45", 123.45},
		{"0.5", 0.5},
		{"100000", 100000.0},
		{"-5.5", -5.5},
		{"-0.01", -0.01},
		{"-100", -100.0},
	}
	for _, tt := range tests {
		got, ok := parseFastFloat([]byte(tt.input))
		if !ok {
			t.Errorf("parseFastFloat(%q) returned false, want true", tt.input)
			continue
		}
		if got != tt.want {
			t.Errorf("parseFastFloat(%q) = %f, want %f", tt.input, got, tt.want)
		}
	}
}

func TestParseFastFloat_Invalid(t *testing.T) {
	tests := []string{
		"",
		"abc",
		"12.34.56",
		"-",
		"N/A",
		" 123.45",
		"123.45 ",
		".",
		"-.",
	}
	for _, input := range tests {
		_, ok := parseFastFloat([]byte(input))
		if ok {
			t.Errorf("parseFastFloat(%q) returned true, want false", input)
		}
	}
}

func TestParseFieldsQuoted_EscapedQuotes(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{`"He said ""hello"""`, []string{`"He said """hello""""`}},
		{`a,"b""c",d`, []string{"a", `"b""c"`, "d"}},
		{`"a""b"`, []string{`"a""b"`}},
	}
	for _, tt := range tests {
		got := parseFieldsQuoted([]byte(tt.input), ',')
		if len(got) != len(tt.want) {
			t.Errorf("parseFieldsQuoted(%q) returned %d fields, want %d", tt.input, len(got), len(tt.want))
			continue
		}
	}
}

func TestParseFieldsQuoted_MultipleFields(t *testing.T) {
	got := parseFieldsQuoted([]byte(`a,"b,c",d`), ',')
	if len(got) != 3 {
		t.Errorf("parseFieldsQuoted returned %d fields, want 3", len(got))
	}
	if got[0] != "a" {
		t.Errorf("field[0] = %q, want %q", got[0], "a")
	}
}

func TestParseCensusFastBytes_NegativeValues(t *testing.T) {
	// Negative age should be rejected
	_, err := parseCensusFastBytes([]byte("-5,M,term,100000,20"), ',')
	if err == nil {
		t.Error("parseCensusFastBytes accepted negative age, want error")
	}

	// Negative sum_assured should be rejected
	_, err = parseCensusFastBytes([]byte("30,M,term,-100000,20"), ',')
	if err == nil {
		t.Error("parseCensusFastBytes accepted negative sum_assured, want error")
	}
}

func TestParseCensusFastBytes_NonNumericFields(t *testing.T) {
	// Non-numeric age
	_, err := parseCensusFastBytes([]byte("abc,M,term,100000,20"), ',')
	if err == nil {
		t.Error("parseCensusFastBytes accepted non-numeric age, want error")
	}

	// Non-numeric sum_assured
	_, err = parseCensusFastBytes([]byte("30,M,term,N/A,20"), ',')
	if err == nil {
		t.Error("parseCensusFastBytes accepted non-numeric sum_assured, want error")
	}

	// Non-numeric term
	_, err = parseCensusFastBytes([]byte("30,M,term,100000,abc"), ',')
	if err == nil {
		t.Error("parseCensusFastBytes accepted non-numeric term, want error")
	}
}
