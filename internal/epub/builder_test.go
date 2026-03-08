package epub_test

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/tpaschalis/patreon-to-epub/internal/epub"
)

func buildEPUB(t *testing.T, post epub.Post) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := epub.Build(&buf, post); err != nil {
		t.Fatalf("Build() error: %v", err)
	}
	return buf.Bytes()
}

func openZIP(t *testing.T, data []byte) *zip.Reader {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("opening EPUB as ZIP: %v", err)
	}
	return r
}

func fileNames(r *zip.Reader) []string {
	names := make([]string, len(r.File))
	for i, f := range r.File {
		names[i] = f.Name
	}
	return names
}

func readFile(t *testing.T, r *zip.Reader, name string) string {
	t.Helper()
	for _, f := range r.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("opening %s: %v", name, err)
			}
			defer rc.Close()
			b, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("reading %s: %v", name, err)
			}
			return string(b)
		}
	}
	t.Fatalf("file %q not found in EPUB; available: %v", name, fileNames(r))
	return ""
}

func TestBuild_requiredFiles(t *testing.T) {
	data := buildEPUB(t, epub.Post{
		Title:       "Test Post",
		Author:      "Test Author",
		PublishedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		SourceURL:   "https://www.patreon.com/posts/test-1",
		Content:     "<p>Hello world</p>",
	})

	r := openZIP(t, data)

	required := []string{
		"mimetype",
		"META-INF/container.xml",
		"OEBPS/content.opf",
		"OEBPS/toc.ncx",
		"OEBPS/content.xhtml",
	}
	names := fileNames(r)
	for _, req := range required {
		found := false
		for _, n := range names {
			if n == req {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("required file %q missing; got: %v", req, names)
		}
	}
}

func TestBuild_mimetype(t *testing.T) {
	data := buildEPUB(t, epub.Post{Title: "T", Author: "A", Content: ""})
	r := openZIP(t, data)

	// mimetype must be the first file and stored uncompressed.
	if len(r.File) == 0 || r.File[0].Name != "mimetype" {
		t.Fatalf("first entry must be 'mimetype', got %v", fileNames(r))
	}
	if r.File[0].Method != zip.Store {
		t.Errorf("mimetype must be stored (uncompressed), got method %d", r.File[0].Method)
	}

	mt := readFile(t, r, "mimetype")
	if mt != "application/epub+zip" {
		t.Errorf("mimetype content = %q, want %q", mt, "application/epub+zip")
	}
}

func TestBuild_contentXHTML(t *testing.T) {
	post := epub.Post{
		Title:       "Hello & World",
		Author:      "A",
		PublishedAt: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		SourceURL:   "https://example.com/post",
		Content:     "<p>some <b>bold</b> text</p>",
	}
	data := buildEPUB(t, post)
	r := openZIP(t, data)
	xhtml := readFile(t, r, "OEBPS/content.xhtml")

	// Must be valid XML-ish.
	if !strings.Contains(xhtml, `<?xml version="1.0"`) {
		t.Error("content.xhtml missing XML declaration")
	}
	if !strings.Contains(xhtml, "Hello &amp; World") {
		t.Error("title not XML-escaped in content.xhtml")
	}
	if !strings.Contains(xhtml, "some <b>bold</b> text") {
		t.Error("post body missing from content.xhtml")
	}
	if !strings.Contains(xhtml, "15 January 2024") {
		t.Error("publication date missing from content.xhtml")
	}
}

func TestBuild_OPFmetadata(t *testing.T) {
	post := epub.Post{
		Title:       "My Post",
		Author:      "Jane Doe",
		PublishedAt: time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC),
		SourceURL:   "https://www.patreon.com/posts/my-post-123",
		Content:     "",
	}
	data := buildEPUB(t, post)
	r := openZIP(t, data)
	opf := readFile(t, r, "OEBPS/content.opf")

	checks := []string{
		"<dc:title>My Post</dc:title>",
		"<dc:creator>Jane Doe</dc:creator>",
		"<dc:date>2024-03-10</dc:date>",
	}
	for _, want := range checks {
		if !strings.Contains(opf, want) {
			t.Errorf("OPF missing %q", want)
		}
	}
}

func TestBuild_xmlEscaping(t *testing.T) {
	post := epub.Post{
		Title:   `Post with <tags> & "quotes"`,
		Author:  "O'Brien",
		Content: "",
	}
	data := buildEPUB(t, post)
	r := openZIP(t, data)
	opf := readFile(t, r, "OEBPS/content.opf")

	if strings.Contains(opf, "<tags>") {
		t.Error("OPF contains unescaped <tags>")
	}
	if !strings.Contains(opf, "&lt;tags&gt;") {
		t.Error("OPF missing escaped tags")
	}
	if !strings.Contains(opf, "&amp;") {
		t.Error("OPF missing escaped ampersand")
	}
}

func TestBuild_noImageURL(t *testing.T) {
	post := epub.Post{
		Title:   "No Image",
		Author:  "A",
		Content: "<p>text only</p>",
	}
	data := buildEPUB(t, post)
	r := openZIP(t, data)

	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "OEBPS/images/") {
			t.Errorf("unexpected image file %q in EPUB with no images", f.Name)
		}
	}
}
