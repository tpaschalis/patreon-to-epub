# patreon-to-epub

A CLI tool that converts Patreon posts you follow into EPUB files, ready to import into an e-reader.

## Status

- [x] Project design & planning
- [x] Patreon API client (identity, memberships, posts)
- [x] CLI selection UI (numbered lists)
- [x] Image downloading & embedding
- [x] EPUB generation
- [x] End-to-end integration (needs real token to validate)

---

## How it works

1. You provide a Patreon API token (see below)
2. The tool lists creators you're currently a patron of
3. You select one or more creators
4. For each creator, you select which posts to convert
5. Each selected post is saved as its own `.epub` file
6. Files are named and organized by creator so you can bulk-import them into your e-reader

---

## Getting a Patreon API Token

1. Go to [https://www.patreon.com/portal/registration/register-clients](https://www.patreon.com/portal/registration/register-clients)
2. Create a new client (name/description don't matter)
3. Copy the **Creator's Access Token** from the client detail page
4. Pass it to the tool via the `--token` flag or the `PATREON_TOKEN` environment variable

> **Note:** This token gives the tool access to your Patreon account (the campaigns you follow and their posts). It is never sent anywhere except to `api.patreon.com`.

---

## Usage

```sh
# Using an env var (recommended)
export PATREON_TOKEN=your_token_here
patreon-to-epub

# Or inline
patreon-to-epub --token your_token_here

# Output directory (default: current directory)
patreon-to-epub --output ~/Books/Patreon
```

### Interactive flow

```
Fetching your memberships...

Creators you follow:
  1. Some Author (42 posts)
  2. Another Creator (17 posts)
  3. Yet Another (9 posts)

Select creators (comma-separated numbers, e.g. 1,3): 1,2

Fetching posts from Some Author...
  1. [2024-11-01] Chapter 42: The Reckoning
  2. [2024-10-15] Chapter 41: A New Dawn
  3. [2024-09-30] World-building notes: The Southern Continent
  ...

Select posts (comma-separated, or 'all'): 1,2

Fetching posts from Another Creator...
  ...

Converting and writing EPUBs...
  ✓ some-author_chapter-42-the-reckoning.epub
  ✓ some-author_chapter-41-a-new-dawn.epub

Done. 2 EPUBs written to ./
```

---

## EPUB structure

Each EPUB contains a single post and follows this layout:

```
post-title.epub
├── mimetype
├── META-INF/
│   └── container.xml
└── OEBPS/
    ├── content.opf        # metadata: title, author, date
    ├── toc.ncx            # navigation
    ├── content.xhtml      # post body (HTML → XHTML)
    └── images/
        └── *.jpg/png/...  # embedded images
```

- Post HTML is cleaned and converted to EPUB-compatible XHTML
- Images are downloaded and embedded (external `src` attributes are rewritten to local paths)
- Audio and video links are preserved as plain text URLs (not embedded)
- EPUB metadata includes: title, author (creator name), publication date, source URL

---

## Project layout

```
patreon-to-epub/
├── cmd/
│   └── patreon-to-epub/
│       └── main.go          # entry point, wires everything together
├── internal/
│   ├── patreon/
│   │   ├── client.go        # HTTP client, auth, pagination
│   │   ├── models.go        # API response types
│   │   └── client_test.go   # integration tests (requires PATREON_TOKEN)
│   ├── epub/
│   │   ├── builder.go       # EPUB file construction
│   │   └── builder_test.go
│   └── ui/
│       └── prompt.go        # numbered-list selection helpers
├── go.mod
└── README.md
```

---

## Dependencies

| Package | Purpose |
|---|---|
| stdlib `net/http` | Patreon API calls and image downloads |
| stdlib `encoding/json` | API response parsing |
| stdlib `archive/zip` | EPUB file construction (EPUB is a ZIP) |
| `golang.org/x/net/html` | HTML parsing for content cleaning |

We deliberately keep dependencies minimal. EPUB generation is done from scratch (it's a ZIP with a defined structure) rather than pulling in a library.

---

## Testing

Integration tests hit the real Patreon API and require a token:

```sh
export PATREON_TOKEN=your_token_here
go test ./...
```

Tests that require the token are skipped automatically if `PATREON_TOKEN` is not set:

```sh
# Skips integration tests
go test ./...

# Runs everything
PATREON_TOKEN=... go test ./...
```

---

## Caveats & known limitations

- Only posts you have access to as a patron will be returned by the API (paywalled posts you haven't unlocked won't appear)
- Patreon's API paginates at 10–20 items per page; the client handles this transparently
- Some posts may contain embedded external content (tweets, YouTube, etc.) — these appear as plain links in the EPUB
- Rate limiting: the tool adds a small delay between requests to stay within Patreon's limits

---

## Known issues / future work

- [ ] OAuth2 flow for better UX (currently requires manual token creation)
- [ ] Audio/video attachment support
- [ ] Cover image for each EPUB (first image in post, or creator avatar)
- [ ] `--since` flag to filter posts by date
- [ ] Batch mode (non-interactive, driven by a config file)
