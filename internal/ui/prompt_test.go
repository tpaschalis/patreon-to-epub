package ui_test

import (
	"strings"
	"testing"

	"github.com/tpaschalis/patreon-to-epub/internal/ui"
)

type item struct{ label string }

func (it item) Label() string { return it.label }

func items(labels ...string) []item {
	out := make([]item, len(labels))
	for i, l := range labels {
		out[i] = item{l}
	}
	return out
}

func TestSelect_single(t *testing.T) {
	got, err := ui.Select("pick:", items("alpha", "beta", "gamma"), nil, strings.NewReader("2\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Label() != "beta" {
		t.Fatalf("expected [beta], got %v", got)
	}
}

func TestSelect_multiple(t *testing.T) {
	got, err := ui.Select("pick:", items("alpha", "beta", "gamma"), nil, strings.NewReader("1,3\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Label() != "alpha" || got[1].Label() != "gamma" {
		t.Fatalf("unexpected selection: %v", got)
	}
}

func TestSelect_all(t *testing.T) {
	src := items("alpha", "beta", "gamma")
	got, err := ui.Select("pick:", src, nil, strings.NewReader("all\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(src) {
		t.Fatalf("expected %d items, got %d", len(src), len(got))
	}
}

func TestSelect_deduplicates(t *testing.T) {
	got, err := ui.Select("pick:", items("alpha", "beta", "gamma"), nil, strings.NewReader("1,1,2\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 items after dedup, got %d", len(got))
	}
}

func TestSelect_invalidNumber(t *testing.T) {
	_, err := ui.Select("pick:", items("alpha", "beta"), nil, strings.NewReader("5\n"))
	if err == nil {
		t.Fatal("expected error for out-of-range selection")
	}
}

func TestSelect_invalidInput(t *testing.T) {
	_, err := ui.Select("pick:", items("alpha", "beta"), nil, strings.NewReader("abc\n"))
	if err == nil {
		t.Fatal("expected error for non-numeric input")
	}
}

func TestSelect_emptyItems(t *testing.T) {
	_, err := ui.Select("pick:", []item{}, nil, strings.NewReader("1\n"))
	if err == nil {
		t.Fatal("expected error for empty item list")
	}
}
