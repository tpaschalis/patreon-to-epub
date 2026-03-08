package patreon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	// internalBaseURL is the base for Patreon's internal (non-OAuth) API,
	// which is the only API layer that allows patrons to read posts.
	internalBaseURL = "https://www.patreon.com/api"

	// apiVersion is the json-api-version required by the internal API.
	apiVersion = "1.0"

	// mobileUA causes Patreon to return higher-resolution media and more
	// complete responses, matching the behaviour of the official Android app.
	mobileUA = "Patreon/126.9.0.15 (Android; Android 14; Scale/2.10)"

	campaignFields = "name,url,patron_count"
	postFields     = "title,content,published_at,url,image"
)

// Client is an authenticated Patreon client using session-cookie auth.
type Client struct {
	sessionID  string
	httpClient *http.Client
}

// NewClient creates a new Client using the given session_id cookie value
// extracted from a logged-in Patreon browser session.
func NewClient(sessionID string) *Client {
	return &Client{
		sessionID: sessionID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// get performs an authenticated GET request and decodes the JSON body into dst.
func (c *Client) get(url string, dst any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.AddCookie(&http.Cookie{Name: "session_id", Value: c.sessionID})
	req.Header.Set("User-Agent", mobileUA)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("invalid or expired session_id (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

// Campaigns returns the list of campaigns (creators) the authenticated patron
// currently follows. It discovers them by walking the patron's post feed.
//
// Pagination stops early once three consecutive pages yield no new campaigns,
// which handles inactive creators without fetching the entire feed history.
func (c *Client) Campaigns() ([]Campaign, error) {
	url := fmt.Sprintf(
		"%s/stream?filter[is_following]=true&json-api-version=%s&include=campaign&fields[campaign]=%s&fields[post]=published_at&page[count]=50",
		internalBaseURL, apiVersion, campaignFields,
	)

	const maxDryPages = 3

	seen := make(map[string]bool)
	var campaigns []Campaign
	dryPages := 0

	for url != "" {
		var resp apiResponse
		if err := c.get(url, &resp); err != nil {
			return nil, fmt.Errorf("fetching stream: %w", err)
		}

		prevLen := len(campaigns)
		for _, inc := range resp.Included {
			if inc.Type == "campaign" && !seen[inc.ID] {
				seen[inc.ID] = true
				campaigns = append(campaigns, Campaign{
					ID:          inc.ID,
					Name:        inc.Attributes.Name,
					URL:         inc.Attributes.URL,
					PatronCount: inc.Attributes.PatronCount,
				})
			}
		}

		if len(campaigns) == prevLen {
			dryPages++
			if dryPages >= maxDryPages {
				break
			}
		} else {
			dryPages = 0
		}

		url = resp.Links.Next
		if url != "" {
			time.Sleep(250 * time.Millisecond)
		}
	}

	return campaigns, nil
}

// Posts returns all posts from the given campaign that the authenticated patron
// has access to. It follows pagination automatically.
func (c *Client) Posts(campaignID string) ([]Post, error) {
	url := fmt.Sprintf(
		"%s/posts?filter[campaign_id]=%s&filter[contains_exclusive_posts]=true&filter[is_draft]=false&sort=-published_at&json-api-version=%s&fields[post]=%s&page[count]=50",
		internalBaseURL, campaignID, apiVersion, postFields,
	)

	var all []Post
	for url != "" {
		var resp apiResponse
		if err := c.get(url, &resp); err != nil {
			return nil, fmt.Errorf("fetching posts (campaign %s): %w", campaignID, err)
		}

		for _, d := range resp.Data {
			p := Post{
				ID:          d.ID,
				Title:       d.Attributes.Title,
				Content:     d.Attributes.Content,
				PublishedAt: d.Attributes.PublishedAt,
				URL:         d.Attributes.URL,
			}
			if d.Attributes.Image != nil {
				if d.Attributes.Image.LargeURL != "" {
					p.ImageURL = d.Attributes.Image.LargeURL
				} else {
					p.ImageURL = d.Attributes.Image.URL
				}
			}
			all = append(all, p)
		}

		url = resp.Links.Next
		if url != "" {
			time.Sleep(250 * time.Millisecond)
		}
	}

	return all, nil
}
