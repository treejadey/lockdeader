package main

import "testing"

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1 KB"},
		{1536, "1.5 KB"},
		{450 * 1024, "450 KB"},
		{24670571, "23.5 MB"},
		{100 * 1024 * 1024, "100 MB"},
		{3 << 30, "3 GB"},
	}

	for _, tt := range tests {
		if got := formatSize(tt.bytes); got != tt.want {
			t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestPlainText(t *testing.T) {
	tests := []struct{ html, want string }{
		{"", ""},
		{"Redesigned Yamato<br><br>Instructions in <span class=\"RedColor\">readme.txt</span><br>stay tuned",
			"Redesigned Yamato\n\nInstructions in readme.txt\nstay tuned"},
		{"<p>One</p><p>Two &amp; three</p>", "One\nTwo & three"},
		{"<style>code { color: red }</style>A&nbsp;&nbsp;B", "A B"},
		{"Lots<br><br><br><br>of space", "Lots\n\nof space"},
	}

	for _, tt := range tests {
		if got := plainText(tt.html); got != tt.want {
			t.Errorf("plainText(%q) = %q, want %q", tt.html, got, tt.want)
		}
	}
}
