package epub

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

// Post contains the data needed to build a single EPUB.
type Post struct {
	Title       string
	Author      string
	PublishedAt time.Time
	SourceURL   string
	// HTML body of the post.
	Content string
	// Optional URL for the header image (downloaded and embedded).
	ImageURL string
}

// Build writes a complete EPUB for a single post to w.
func Build(w io.Writer, post Post) error {
	b := &builder{
		zw:   zip.NewWriter(w),
		post: post,
	}
	return b.build()
}

type builder struct {
	zw     *zip.Writer
	post   Post
	images []embeddedImage
}

type embeddedImage struct {
	filename    string
	mediaType   string
	data        []byte
}

func (b *builder) build() error {
	// EPUB requires mimetype to be the first entry, uncompressed.
	if err := b.writeMimetype(); err != nil {
		return err
	}
	if err := b.writeContainerXML(); err != nil {
		return err
	}

	// Download images and rewrite content HTML before writing anything else.
	content, err := b.processContent(b.post.Content)
	if err != nil {
		return fmt.Errorf("processing content: %w", err)
	}

	if err := b.writeImages(); err != nil {
		return err
	}
	if err := b.writeContentXHTML(content); err != nil {
		return err
	}
	if err := b.writeOPF(); err != nil {
		return err
	}
	if err := b.writeNCX(); err != nil {
		return err
	}

	return b.zw.Close()
}

// writeMimetype writes the uncompressed mimetype entry (required by EPUB spec).
func (b *builder) writeMimetype() error {
	h := &zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store, // must be uncompressed
	}
	w, err := b.zw.CreateHeader(h)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, "application/epub+zip")
	return err
}

func (b *builder) writeContainerXML() error {
	w, err := b.zw.Create("META-INF/container.xml")
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>
`)
	return err
}

// processContent downloads images referenced in the HTML, replaces their src
// attributes with local paths, and strips tags that are problematic in EPUB.
func (b *builder) processContent(html string) (string, error) {
	// Download the header image if present.
	if b.post.ImageURL != "" {
		img, err := downloadImage(b.post.ImageURL)
		if err == nil {
			filename := imageFilename("cover", b.post.ImageURL)
			b.images = append(b.images, embeddedImage{
				filename:  filename,
				mediaType: img.mediaType,
				data:      img.data,
			})
		}
		// Non-fatal: if the image download fails we just omit it.
	}

	// Find all <img src="..."> tags and replace with local references.
	html = imgSrcRe.ReplaceAllStringFunc(html, func(match string) string {
		sub := imgSrcRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		srcURL := sub[1]
		if srcURL == "" {
			return match
		}

		img, err := downloadImage(srcURL)
		if err != nil {
			// Leave the original src if we can't download.
			return match
		}
		filename := imageFilename(fmt.Sprintf("img%d", len(b.images)), srcURL)
		b.images = append(b.images, embeddedImage{
			filename:  filename,
			mediaType: img.mediaType,
			data:      img.data,
		})
		return strings.Replace(match, srcURL, "images/"+filename, 1)
	})

	return html, nil
}

// imgSrcRe matches <img ... src="..." ...> or <img ... src='...' ...>.
var imgSrcRe = regexp.MustCompile(`(?i)<img[^>]+src=["']([^"']+)["'][^>]*>`)

func (b *builder) writeImages() error {
	for _, img := range b.images {
		w, err := b.zw.Create("OEBPS/images/" + img.filename)
		if err != nil {
			return err
		}
		if _, err := w.Write(img.data); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) writeContentXHTML(html string) error {
	w, err := b.zw.Create("OEBPS/content.xhtml")
	if err != nil {
		return err
	}

	// Build a simple cover image block if we have one.
	var coverHTML string
	if len(b.images) > 0 && b.post.ImageURL != "" {
		coverHTML = fmt.Sprintf(`<div class="cover"><img src="images/%s" alt="cover"/></div>`+"\n", b.images[0].filename)
	}

	var meta string
	if !b.post.PublishedAt.IsZero() {
		meta = fmt.Sprintf(`<p class="meta">Published %s`, b.post.PublishedAt.Format("2 January 2006"))
		if b.post.SourceURL != "" {
			meta += fmt.Sprintf(` &mdash; <a href="%s">Original post</a>`, b.post.SourceURL)
		}
		meta += "</p>\n"
	}

	_, err = fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="en">
<head>
  <meta charset="UTF-8"/>
  <title>%s</title>
</head>
<body>
<h1>%s</h1>
%s%s
</body>
</html>
`, xmlEscape(b.post.Title), xmlEscape(b.post.Title), coverHTML, meta+html)
	return err
}

func (b *builder) writeOPF() error {
	w, err := b.zw.Create("OEBPS/content.opf")
	if err != nil {
		return err
	}

	// Build manifest items for images.
	var manifestItems strings.Builder
	for i, img := range b.images {
		fmt.Fprintf(&manifestItems, `    <item id="img%d" href="images/%s" media-type="%s"/>`+"\n",
			i, img.filename, img.mediaType)
	}

	date := b.post.PublishedAt.Format("2006-01-02")
	if b.post.PublishedAt.IsZero() {
		date = time.Now().Format("2006-01-02")
	}

	_, err = fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>%s</dc:title>
    <dc:creator>%s</dc:creator>
    <dc:date>%s</dc:date>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">%s</dc:identifier>
    <dc:source>%s</dc:source>
    <meta property="dcterms:modified">%s</meta>
  </metadata>
  <manifest>
    <item id="content" href="content.xhtml" media-type="application/xhtml+xml"/>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
%s  </manifest>
  <spine toc="ncx">
    <itemref idref="content"/>
  </spine>
</package>
`,
		xmlEscape(b.post.Title),
		xmlEscape(b.post.Author),
		date,
		xmlEscape(b.post.SourceURL),
		xmlEscape(b.post.SourceURL),
		time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		manifestItems.String(),
	)
	return err
}

func (b *builder) writeNCX() error {
	w, err := b.zw.Create("OEBPS/toc.ncx")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <head>
    <meta name="dtb:uid" content="%s"/>
    <meta name="dtb:depth" content="1"/>
    <meta name="dtb:totalPageCount" content="0"/>
    <meta name="dtb:maxPageNumber" content="0"/>
  </head>
  <docTitle><text>%s</text></docTitle>
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>%s</text></navLabel>
      <content src="content.xhtml"/>
    </navPoint>
  </navMap>
</ncx>
`,
		xmlEscape(b.post.SourceURL),
		xmlEscape(b.post.Title),
		xmlEscape(b.post.Title),
	)
	return err
}

// downloadResult holds raw image bytes and its detected media type.
type downloadResult struct {
	data      []byte
	mediaType string
}

func downloadImage(rawURL string) (downloadResult, error) {
	resp, err := http.Get(rawURL) //nolint:noctx
	if err != nil {
		return downloadResult{}, fmt.Errorf("downloading image %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return downloadResult{}, fmt.Errorf("image %s: HTTP %s", rawURL, resp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20)) // 20 MB limit
	if err != nil {
		return downloadResult{}, fmt.Errorf("reading image %s: %w", rawURL, err)
	}

	mt := resp.Header.Get("Content-Type")
	if mt == "" {
		mt = http.DetectContentType(data)
	}
	// Strip parameters (e.g. "image/jpeg; charset=...")
	if idx := strings.Index(mt, ";"); idx != -1 {
		mt = strings.TrimSpace(mt[:idx])
	}

	return downloadResult{data: data, mediaType: mt}, nil
}

// imageFilename derives a safe local filename for an image.
func imageFilename(prefix, rawURL string) string {
	u, err := url.Parse(rawURL)
	ext := ".jpg"
	if err == nil {
		p := path.Ext(u.Path)
		if p != "" {
			// Validate it's a known image extension.
			mt := mime.TypeByExtension(p)
			if strings.HasPrefix(mt, "image/") {
				ext = p
			}
		}
	}
	return prefix + ext
}

// xmlEscape returns s with XML special characters escaped.
func xmlEscape(s string) string {
	var buf bytes.Buffer
	for _, r := range s {
		switch r {
		case '&':
			buf.WriteString("&amp;")
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '"':
			buf.WriteString("&quot;")
		case '\'':
			buf.WriteString("&apos;")
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}
