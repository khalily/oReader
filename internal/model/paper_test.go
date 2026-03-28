package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaperStatusConstants(t *testing.T) {
	assert.Equal(t, "pending", PaperStatusPending)
	assert.Equal(t, "processing", PaperStatusProcessing)
	assert.Equal(t, "completed", PaperStatusCompleted)
	assert.Equal(t, "failed", PaperStatusFailed)
}

func TestPaper_GenerateID(t *testing.T) {
	paper := &Paper{}
	err := paper.GenerateID()
	require.NoError(t, err)
	assert.NotEmpty(t, paper.ID)
	assert.Len(t, paper.ID, 36) // UUID v7 format
}

func TestPaperCollection_GenerateID(t *testing.T) {
	collection := &PaperCollection{}
	err := collection.GenerateID()
	require.NoError(t, err)
	assert.NotEmpty(t, collection.ID)
	assert.Len(t, collection.ID, 36)
}

func TestPaper_AuthorsJSON(t *testing.T) {
	paper := &Paper{
		Authors: `["Alice","Bob"]`,
	}
	var authors []string
	err := json.Unmarshal([]byte(paper.Authors), &authors)
	require.NoError(t, err)
	assert.Equal(t, []string{"Alice", "Bob"}, authors)
}

func TestPaper_KeywordsJSON(t *testing.T) {
	paper := &Paper{
		Keywords: `["deep learning","transformer"]`,
	}
	var keywords []string
	err := json.Unmarshal([]byte(paper.Keywords), &keywords)
	require.NoError(t, err)
	assert.Equal(t, []string{"deep learning", "transformer"}, keywords)
}
