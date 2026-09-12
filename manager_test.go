package main

import "testing"

func TestFindCategoryByName(t *testing.T) {
	t.Run("known category", func(t *testing.T) {
		category, err := findCategoryByName("songs")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if category == nil {
			t.Fatal("expected a non-nil category")
		}
		if *category != CategorySongs {
			t.Errorf("category = %+v, want %+v", *category, CategorySongs)
		}
	})

	t.Run("unknown category", func(t *testing.T) {
		category, err := findCategoryByName("nao-existe")
		if err == nil {
			t.Fatal("expected an error for an unknown category")
		}
		if category != nil {
			t.Errorf("expected nil category, got %+v", *category)
		}
	})
}
