package patreon

import "time"

// apiResponse is the common JSON:API envelope returned by the internal Patreon API
// for both the /api/stream and /api/posts endpoints.
type apiResponse struct {
	Data     []postData `json:"data"`
	Included []included `json:"included"`
	Links    struct {
		Next string `json:"next"`
	} `json:"links"`
}

// included is a polymorphic resource in the JSON:API "included" array.
// We only decode the fields we care about.
type included struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Attributes includedAttributes `json:"attributes"`
}

type includedAttributes struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	PatronCount int    `json:"patron_count"`
}

// Campaign is the parsed, usable form of a creator's campaign.
type Campaign struct {
	ID          string
	Name        string
	URL         string
	PatronCount int
}

// postData is a single post resource in a JSON:API response.
type postData struct {
	ID         string         `json:"id"`
	Attributes postAttributes `json:"attributes"`
}

type postAttributes struct {
	Title       string     `json:"title"`
	Content     string     `json:"content"` // HTML
	PublishedAt time.Time  `json:"published_at"`
	URL         string     `json:"url"`
	Image       *postImage `json:"image"`
}

type postImage struct {
	URL      string `json:"url"`
	LargeURL string `json:"large_url"`
}

// Post is the parsed, usable form of a Patreon post.
type Post struct {
	ID          string
	Title       string
	Content     string // HTML
	PublishedAt time.Time
	URL         string
	ImageURL    string // may be empty
}
