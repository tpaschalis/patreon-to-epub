package patreon_test

import (
	"os"
	"testing"
	"time"

	"github.com/tpaschalis/patreon-to-epub/internal/patreon"
)

func sessionID(t *testing.T) string {
	t.Helper()
	sid := os.Getenv("PATREON_SESSION_ID")
	if sid == "" {
		t.Skip("PATREON_SESSION_ID not set; skipping integration test")
	}
	return sid
}

func TestCampaigns(t *testing.T) {
	c := patreon.NewClient(sessionID(t))

	campaigns, err := c.Campaigns()
	if err != nil {
		t.Fatalf("Campaigns() error: %v", err)
	}

	if len(campaigns) == 0 {
		t.Log("no campaigns returned — are you currently a patron of anyone?")
		return
	}

	t.Logf("found %d campaign(s):", len(campaigns))
	for _, camp := range campaigns {
		if camp.ID == "" {
			t.Errorf("campaign missing ID: %+v", camp)
		}
		if camp.Name == "" {
			t.Errorf("campaign %s missing Name", camp.ID)
		}
		t.Logf("  [%s] %s (%s)", camp.ID, camp.Name, camp.URL)
	}
}

func TestPosts(t *testing.T) {
	c := patreon.NewClient(sessionID(t))

	campaigns, err := c.Campaigns()
	if err != nil {
		t.Fatalf("Campaigns() error: %v", err)
	}
	if len(campaigns) == 0 {
		t.Skip("no campaigns to test posts with")
	}

	// Test against the first campaign only to keep the test fast.
	camp := campaigns[0]
	t.Logf("fetching posts for campaign %q (ID: %s)", camp.Name, camp.ID)

	posts, err := c.Posts(camp.ID)
	if err != nil {
		t.Fatalf("Posts() error: %v", err)
	}

	if len(posts) == 0 {
		t.Log("no posts returned — the creator may have no accessible posts")
		return
	}

	t.Logf("found %d post(s):", len(posts))
	for i, p := range posts {
		if i >= 5 {
			t.Logf("  ... and %d more", len(posts)-5)
			break
		}
		if p.ID == "" {
			t.Errorf("post missing ID: %+v", p)
		}
		if p.PublishedAt.IsZero() {
			t.Errorf("post %s has zero PublishedAt", p.ID)
		}
		if p.PublishedAt.After(time.Now().Add(24 * time.Hour)) {
			t.Errorf("post %s has future PublishedAt: %v", p.ID, p.PublishedAt)
		}
		t.Logf("  [%s] %s — %s", p.PublishedAt.Format("2006-01-02"), p.Title, p.URL)
	}
}

func TestInvalidSession(t *testing.T) {
	c := patreon.NewClient("invalid-session-for-testing")
	_, err := c.Campaigns()
	if err == nil {
		t.Fatal("expected error for invalid session_id, got nil")
	}
	t.Logf("got expected error: %v", err)
}
