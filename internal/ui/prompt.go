package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Item is anything that can be displayed in a selection list.
type Item interface {
	Label() string
}

// Select displays a numbered list of items and asks the user to pick one or
// more by number. Passing "all" selects every item.
//
// out and in are used for output/input; pass nil to use os.Stdout/os.Stdin.
func Select[T Item](prompt string, items []T, out io.Writer, in io.Reader) ([]T, error) {
	if out == nil {
		out = os.Stdout
	}
	if in == nil {
		in = os.Stdin
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no items to select from")
	}

	fmt.Fprintln(out, prompt)
	for i, item := range items {
		fmt.Fprintf(out, "  %d. %s\n", i+1, item.Label())
	}
	fmt.Fprint(out, "\nSelect (comma-separated numbers, or 'all'): ")

	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return nil, fmt.Errorf("reading input: %w", scanner.Err())
	}
	raw := strings.TrimSpace(scanner.Text())

	if strings.ToLower(raw) == "all" {
		return items, nil
	}

	var selected []T
	seen := make(map[int]bool)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 || n > len(items) {
			return nil, fmt.Errorf("invalid selection %q: must be a number between 1 and %d", part, len(items))
		}
		if !seen[n] {
			seen[n] = true
			selected = append(selected, items[n-1])
		}
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf("no items selected")
	}
	return selected, nil
}
