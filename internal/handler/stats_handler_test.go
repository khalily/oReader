package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"oreader/internal/service"
)

// mockStatsService is a mock implementation of StatsService for testing
type mockStatsService struct {
	getUserStatsFunc func(ctx context.Context, userID string) (*service.UserStats, error)
}

func (m *mockStatsService) GetUserStats(ctx context.Context, userID string) (*service.UserStats, error) {
	if m.getUserStatsFunc != nil {
		return m.getUserStatsFunc(ctx, userID)
	}
	return &service.UserStats{}, nil
}

func TestStatsHandler_GetStats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "test-user-id"
	expectedStats := &service.UserStats{
		Total:   100,
		Unread:  25,
		Starred: 10,
		Today:   5,
	}

	mockService := &mockStatsService{
		getUserStatsFunc: func(ctx context.Context, uid string) (*service.UserStats, error) {
			assert.Equal(t, userID, uid)
			return expectedStats, nil
		},
	}
	handler := NewStatsHandler(mockService)

	router := gin.New()
	router.GET("/stats", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.GetStats)

	req := httptest.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var response service.UserStats
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, int64(100), response.Total)
	assert.Equal(t, int64(25), response.Unread)
	assert.Equal(t, int64(10), response.Starred)
	assert.Equal(t, int64(5), response.Today)
}

func TestStatsHandler_GetStats_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockService := &mockStatsService{}
	handler := NewStatsHandler(mockService)

	router := gin.New()
	router.GET("/stats", handler.GetStats)

	req := httptest.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestStatsHandler_GetStats_ZeroStats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := "new-user"

	mockService := &mockStatsService{
		getUserStatsFunc: func(ctx context.Context, uid string) (*service.UserStats, error) {
			return &service.UserStats{
				Total:   0,
				Unread:  0,
				Starred: 0,
				Today:   0,
			}, nil
		},
	}
	handler := NewStatsHandler(mockService)

	router := gin.New()
	router.GET("/stats", func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}, handler.GetStats)

	req := httptest.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var response service.UserStats
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, int64(0), response.Total)
	assert.Equal(t, int64(0), response.Unread)
	assert.Equal(t, int64(0), response.Starred)
	assert.Equal(t, int64(0), response.Today)
}
