package model

import "testing"

func TestCategoryTypeValues(t *testing.T) {
	validTypes := []string{CategoryTypeFeed, CategoryTypePaper}
	for _, ct := range validTypes {
		if ct != "feed" && ct != "paper" {
			t.Errorf("unexpected category type: %s", ct)
		}
	}
}

func TestCategoryGenerateID(t *testing.T) {
	c := &Category{Name: "Test", Type: CategoryTypeFeed, UserID: "user-1"}
	if err := c.GenerateID(); err != nil {
		t.Fatalf("GenerateID failed: %v", err)
	}
	if c.ID == "" {
		t.Error("ID should not be empty")
	}
}
