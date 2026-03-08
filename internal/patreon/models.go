package patreon

import "time"

// JSON:API envelope types

type resourceID struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type relationship struct {
	Data []resourceID `json:"data"`
}

type relationshipSingle struct {
	Data resourceID `json:"data"`
}

// Identity (current user)

type identityResponse struct {
	Data     identityData `json:"data"`
	Included []included   `json:"included"`
}

type identityData struct {
	ID            string             `json:"id"`
	Attributes    userAttributes     `json:"attributes"`
	Relationships identityRelations  `json:"relationships"`
}

type userAttributes struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

type identityRelations struct {
	Memberships relationship `json:"memberships"`
}

// Included resources (members and campaigns come back in the same array)

type included struct {
	ID            string              `json:"id"`
	Type          string              `json:"type"`
	Attributes    includedAttributes  `json:"attributes"`
	Relationships memberRelations     `json:"relationships"`
}

type includedAttributes struct {
	// campaign fields
	Name         string `json:"name"`
	URL          string `json:"url"`
	PatronCount  int    `json:"patron_count"`

	// member fields (nothing extra needed)
}

type memberRelations struct {
	Campaign relationshipSingle `json:"campaign"`
}

// Campaign is the parsed, usable form of a creator's campaign.
type Campaign struct {
	ID          string
	Name        string
	URL         string
	PatronCount int
}

// Posts

type postsResponse struct {
	Data  []postData `json:"data"`
	Links struct {
		Next string `json:"next"`
	} `json:"links"`
	Meta struct {
		Pagination struct {
			Total int `json:"total"`
		} `json:"pagination"`
	} `json:"meta"`
}

type postData struct {
	ID         string         `json:"id"`
	Attributes postAttributes `json:"attributes"`
}

type postAttributes struct {
	Title       string    `json:"title"`
	Content     string    `json:"content"` // HTML
	PublishedAt time.Time `json:"published_at"`
	URL         string    `json:"url"`
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
