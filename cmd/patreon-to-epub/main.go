package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/tpaschalis/patreon-to-epub/internal/epub"
	"github.com/tpaschalis/patreon-to-epub/internal/patreon"
	"github.com/tpaschalis/patreon-to-epub/internal/ui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	token := flag.String("token", "", "Patreon API token (or set PATREON_TOKEN)")
	outDir := flag.String("output", ".", "Directory to write EPUB files into")
	flag.Parse()

	if *token == "" {
		*token = os.Getenv("PATREON_TOKEN")
	}
	if *token == "" {
		return fmt.Errorf("a Patreon API token is required (--token or PATREON_TOKEN)")
	}

	client := patreon.NewClient(*token)

	// --- Select creators ---
	fmt.Println("Fetching your memberships...")
	campaigns, err := client.Campaigns()
	if err != nil {
		return fmt.Errorf("fetching campaigns: %w", err)
	}
	if len(campaigns) == 0 {
		return fmt.Errorf("you don't appear to be a patron of any creator")
	}

	selectedCampaigns, err := ui.Select("\nCreators you follow:", campaignItems(campaigns), os.Stdout, os.Stdin)
	if err != nil {
		return fmt.Errorf("selecting creators: %w", err)
	}

	// --- For each creator, select posts and convert ---
	var totalWritten int
	for _, camp := range selectedCampaigns {
		fmt.Printf("\nFetching posts from %s...\n", camp.c.Name)
		posts, err := client.Posts(camp.c.ID)
		if err != nil {
			return fmt.Errorf("fetching posts for %s: %w", camp.c.Name, err)
		}
		if len(posts) == 0 {
			fmt.Printf("  No accessible posts found for %s.\n", camp.c.Name)
			continue
		}

		selectedPosts, err := ui.Select(
			fmt.Sprintf("\nPosts from %s:", camp.c.Name),
			postItems(posts),
			os.Stdout,
			os.Stdin,
		)
		if err != nil {
			return fmt.Errorf("selecting posts for %s: %w", camp.c.Name, err)
		}

		fmt.Println("\nConverting and writing EPUBs...")
		for _, p := range selectedPosts {
			filename := epubFilename(*outDir, camp.c.Name, p.p.Title, p.p.PublishedAt)
			if err := writeEPUB(filename, camp.c.Name, p.p); err != nil {
				return fmt.Errorf("writing %s: %w", filename, err)
			}
			fmt.Printf("  ✓ %s\n", filepath.Base(filename))
			totalWritten++
		}
	}

	fmt.Printf("\nDone. %d EPUB(s) written to %s\n", totalWritten, *outDir)
	return nil
}

func writeEPUB(filename, author string, p patreon.Post) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return epub.Build(f, epub.Post{
		Title:       p.Title,
		Author:      author,
		PublishedAt: p.PublishedAt,
		SourceURL:   p.URL,
		Content:     p.Content,
		ImageURL:    p.ImageURL,
	})
}

// epubFilename builds a safe output path: <outDir>/<author>_<date>_<slug>.epub
func epubFilename(outDir, author, title string, date time.Time) string {
	slug := slugify(author) + "_" + date.Format("2006-01-02") + "_" + slugify(title)
	if len(slug) > 180 {
		slug = slug[:180]
	}
	return filepath.Join(outDir, slug+".epub")
}

var nonAlphanumRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlphanumRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// --- ui.Item adapters ---

type campaignItem struct{ c patreon.Campaign }

func (ci campaignItem) Label() string { return ci.c.Name }

func campaignItems(cs []patreon.Campaign) []campaignItem {
	out := make([]campaignItem, len(cs))
	for i, c := range cs {
		out[i] = campaignItem{c}
	}
	return out
}

type postItem struct{ p patreon.Post }

func (pi postItem) Label() string {
	date := pi.p.PublishedAt.Format("2006-01-02")
	title := pi.p.Title
	if title == "" {
		title = "(untitled)"
	}
	return fmt.Sprintf("[%s] %s", date, title)
}

func postItems(ps []patreon.Post) []postItem {
	out := make([]postItem, len(ps))
	for i, p := range ps {
		out[i] = postItem{p}
	}
	return out
}
