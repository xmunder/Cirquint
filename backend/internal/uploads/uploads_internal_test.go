package uploads

import "testing"

func TestMimeTypeFromFilenameVariants(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{name: "png", filename: "diagram.png", want: "image/png"},
		{name: "jpeg", filename: "diagram.jpeg", want: "image/jpeg"},
		{name: "jpg", filename: "diagram.jpg", want: "image/jpeg"},
		{name: "webp", filename: "diagram.webp", want: "image/webp"},
		{name: "default", filename: "diagram.gif", want: "application/gif"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mimeTypeFromFilename(tt.filename); got != tt.want {
				t.Fatalf("mimeTypeFromFilename(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}
