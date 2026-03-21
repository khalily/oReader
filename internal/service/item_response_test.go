package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"oreader/internal/model"
)

// =============================================================================
// TDD RED PHASE: These tests verify the EXPECTED JSON response structure.
// They will FAIL with the current implementation because ItemWithState
// currently uses flat fields instead of nested user_state and feed objects.
// =============================================================================

// TestItemWithState_JSON_HasNestedUserState verifies that JSON output has
// a nested "user_state" object containing is_starred, is_read, read_at.
//
// EXPECTED: {"user_state": {"item_id": "...", "is_starred": true, "is_read": false, "read_at": "..."}}
// CURRENT (WRONG): {"is_starred": true, "is_read": false, "read_at": "..."}
func TestItemWithState_JSON_HasNestedUserState(t *testing.T) {
	// Create an item with user state
	item := &model.Item{
		Base: model.Base{ID: "item-1"},
		FeedID: "feed-1",
		Title: "Test Item",
		Link: "https://example.com/item",
	}

	// Create ItemWithState with user state
	iws := &ItemWithState{
		Item: item,
		// These should be nested under user_state, not flat fields
	}

	// Serialize to JSON
	data, err := json.Marshal(iws)
	if err != nil {
		t.Fatalf("Failed to marshal ItemWithState: %v", err)
	}

	jsonStr := string(data)
	t.Logf("JSON output: %s", jsonStr)

	// Verify JSON has nested user_state object (this will FAIL with current impl)
	// The JSON should contain "user_state":{...} not flat "is_starred"
	if !strings.Contains(jsonStr, `"user_state"`) {
		t.Errorf("Expected JSON to contain nested 'user_state' object, got: %s", jsonStr)
	}

	// Verify is_starred is INSIDE user_state, not at top level
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check that is_starred is NOT at the top level (current wrong behavior)
	if _, exists := result["is_starred"]; exists {
		t.Errorf("JSON should NOT have 'is_starred' at top level, should be nested in 'user_state'")
	}

	// Check that user_state exists and is an object
	userState, exists := result["user_state"]
	if !exists {
		t.Fatalf("JSON must have 'user_state' field, got: %s", jsonStr)
	}

	// user_state should be an object or null
	if userState != nil {
		userStateMap, ok := userState.(map[string]interface{})
		if !ok {
			t.Errorf("user_state should be an object, got: %T", userState)
		} else {
			// Verify user_state contains the expected fields
			if _, exists := userStateMap["is_starred"]; !exists {
				t.Error("user_state should contain 'is_starred'")
			}
			if _, exists := userStateMap["is_read"]; !exists {
				t.Error("user_state should contain 'is_read'")
			}
		}
	}
}

// TestItemWithState_JSON_HasNestedFeedObject verifies that JSON output has
// a nested "feed" object with id and title, not just "feed_title" string.
//
// EXPECTED: {"feed": {"id": "...", "title": "..."}}
// CURRENT (WRONG): {"feed_title": "..."}
func TestItemWithState_JSON_HasNestedFeedObject(t *testing.T) {
	// Create an item with feed
	item := &model.Item{
		Base: model.Base{ID: "item-1"},
		FeedID: "feed-1",
		Title: "Test Item",
		Link: "https://example.com/item",
		Feed: &model.Feed{
			Base:  model.Base{ID: "feed-1"},
			Title: "Test Feed",
		},
	}

	// Create ItemWithState with Feed populated
	iws := &ItemWithState{
		Item: item,
		Feed: buildFeedResponse(item.Feed),
	}

	// Serialize to JSON
	data, err := json.Marshal(iws)
	if err != nil {
		t.Fatalf("Failed to marshal ItemWithState: %v", err)
	}

	jsonStr := string(data)
	t.Logf("JSON output: %s", jsonStr)

	// Verify JSON has nested feed object (this will FAIL with current impl)
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check that feed_title is NOT present (current wrong behavior)
	if _, exists := result["feed_title"]; exists {
		t.Errorf("JSON should NOT have 'feed_title' at top level, should have nested 'feed' object")
	}

	// Check that feed exists and is an object with id and title
	feed, exists := result["feed"]
	if !exists {
		t.Fatalf("JSON must have 'feed' field, got: %s", jsonStr)
	}

	feedMap, ok := feed.(map[string]interface{})
	if !ok {
		t.Fatalf("'feed' should be an object, got: %T", feed)
	}

	// Verify feed contains id and title
	if _, exists := feedMap["id"]; !exists {
		t.Error("feed should contain 'id'")
	}
	if _, exists := feedMap["title"]; !exists {
		t.Error("feed should contain 'title'")
	}
}

// TestItemWithState_JSON_UserStateCanBeNull verifies that user_state can be null
// when the user has no interaction with the item.
//
// EXPECTED: {"user_state": null} when no state exists
// CURRENT (WRONG): No user_state field at all, or flat fields with default values
func TestItemWithState_JSON_UserStateCanBeNull(t *testing.T) {
	// Create an item WITHOUT user state
	item := &model.Item{
		Base: model.Base{ID: "item-1"},
		FeedID: "feed-1",
		Title: "Test Item",
		Link: "https://example.com/item",
	}

	// Create ItemWithState without user state (UserState should be nil)
	iws := &ItemWithState{
		Item: item,
	}

	// Serialize to JSON
	data, err := json.Marshal(iws)
	if err != nil {
		t.Fatalf("Failed to marshal ItemWithState: %v", err)
	}

	jsonStr := string(data)
	t.Logf("JSON output: %s", jsonStr)

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// user_state field must exist (can be null)
	userState, exists := result["user_state"]
	if !exists {
		t.Errorf("JSON must have 'user_state' field even when null, got: %s", jsonStr)
	}

	// user_state should be null when no state exists
	if userState != nil {
		t.Errorf("user_state should be null when no state exists, got: %v", userState)
	}
}

// TestItemWithState_JSON_CompleteStructure verifies the complete expected structure
func TestItemWithState_JSON_CompleteStructure(t *testing.T) {
	// Create an item with all fields
	now := time.Now()
	item := &model.Item{
		Base: model.Base{ID: "item-1"},
		FeedID: "feed-1",
		Title: "Test Item",
		Link: "https://example.com/item",
		Description: "Test description",
		Content: "<p>Content</p>",
		PubDate: &now,
		Creator: "Author",
		Feed: &model.Feed{
			Base:  model.Base{ID: "feed-1"},
			Title: "Test Feed",
		},
	}

	// Create ItemWithState with Feed populated
	iws := &ItemWithState{
		Item: item,
		Feed: buildFeedResponse(item.Feed),
	}

	// Serialize to JSON
	data, err := json.Marshal(iws)
	if err != nil {
		t.Fatalf("Failed to marshal ItemWithState: %v", err)
	}

	jsonStr := string(data)
	t.Logf("JSON output: %s", jsonStr)

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify the expected structure:
	// 1. Item fields at top level (id, title, link, etc.)
	if result["id"] != "item-1" {
		t.Errorf("Expected id='item-1', got: %v", result["id"])
	}
	if result["title"] != "Test Item" {
		t.Errorf("Expected title='Test Item', got: %v", result["title"])
	}

	// 2. feed object (not feed_title string)
	if _, exists := result["feed"]; !exists {
		t.Error("Missing 'feed' object")
	}
	if _, exists := result["feed_title"]; exists {
		t.Error("Should NOT have 'feed_title' at top level")
	}

	// 3. user_state object (not flat is_starred/is_read)
	if _, exists := result["user_state"]; !exists {
		t.Error("Missing 'user_state' field")
	}
	if _, exists := result["is_starred"]; exists {
		t.Error("Should NOT have 'is_starred' at top level")
	}
	if _, exists := result["is_read"]; exists {
		t.Error("Should NOT have 'is_read' at top level")
	}
}