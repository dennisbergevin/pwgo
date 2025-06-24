package main

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
)

func TestReinsertInOriginalPosition(t *testing.T) {
	// Define original list of items in correct order
	original := []item{
		{title: "A", description: "descA"},
		{title: "B", description: "descB"},
		{title: "C", description: "descC"},
	}

	// Current list
	cur := []list.Item{
		item{title: "B", description: "descB"},
		item{title: "C", description: "descC"},
	}

	// Item to reinsert
	sel := item{title: "A", description: "descA"}

	// Create the list model and set items
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.SetItems(cur)

	// Reinsert into the correct position
	reinsertInOriginalPosition(sel, &l, original)

	expected := []item{
		{title: "A", description: "descA"},
		{title: "B", description: "descB"},
		{title: "C", description: "descC"},
	}

	result := l.Items()
	if len(result) != len(expected) {
		t.Fatalf("expected list length %d, got %d", len(expected), len(result))
	}

	for i := range result {
		got := result[i].(item)
		want := expected[i]
		if got.title != want.title || got.description != want.description {
			t.Errorf("item at index %d - expected %+v, got %+v", i, want, got)
		}
	}
}
