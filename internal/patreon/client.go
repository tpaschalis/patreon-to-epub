package patreon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	baseURL = "https://www.patreon.com/api/oauth2/v2"

	// Fields we request from the API.
	campaignFields = "name,url,patron_count"
	postFields     = "title,content,published_at,url,image"
	userFields     = "full_name,email"
)

// Client is an authenticated Patreon API v2 client.
type Client struct {
	token      string
	httpClient *http.Client
}

// NewClient creates a new Client using the given API token.
func NewClient(token string) *Client {
	return &Client{
		token: token,
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
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", "patreon-to-epub/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("invalid or expired API token (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

// Campaigns returns the list of campaigns (creators) the authenticated user
// is currently a patron of.
func (c *Client) Campaigns() ([]Campaign, error) {
	url := fmt.Sprintf(
		"%s/identity?include=memberships.campaign&fields[campaign]=%s&fields[user]=%s",
		baseURL, campaignFields, userFields,
	)

	var resp identityResponse
	if err := c.get(url, &resp); err != nil {
		return nil, fmt.Errorf("fetching identity: %w", err)
	}

	// Build a map of campaign ID → Campaign from the included resources.
	campaigns := make(map[string]Campaign)
	for _, inc := range resp.Included {
		if inc.Type == "campaign" {
			campaigns[inc.ID] = Campaign{
				ID:          inc.ID,
				Name:        inc.Attributes.Name,
				URL:         inc.Attributes.URL,
				PatronCount: inc.Attributes.PatronCount,
			}
		}
	}

	// Walk memberships to collect only the campaigns we're a patron of,
	// preserving the order returned by the API.
	seen := make(map[string]bool)
	var result []Campaign
	for _, memberRef := range resp.Data.Relationships.Memberships.Data {
		// Find the member in included to get its campaign relationship.
		for _, inc := range resp.Included {
			if inc.Type == "member" && inc.ID == memberRef.ID {
				cid := inc.Relationships.Campaign.Data.ID
				if c, ok := campaigns[cid]; ok && !seen[cid] {
					seen[cid] = true
					result = append(result, c)
				}
			}
		}
	}

	return result, nil
}

// Posts returns all posts from the given campaign that the authenticated user
// has access to. It follows pagination automatically.
func (c *Client) Posts(campaignID string) ([]Post, error) {
	url := fmt.Sprintf(
		"%s/campaigns/%s/posts?fields[post]=%s&page[count]=20&sort=-published_at",
		baseURL, campaignID, postFields,
	)

	var all []Post
	for url != "" {
		var resp postsResponse
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
			// Be polite to the API.
			time.Sleep(250 * time.Millisecond)
		}
	}

	return all, nil
}
