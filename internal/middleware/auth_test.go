package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUserID(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	userID, exists := GetUserID(req)
	assert.False(t, exists)
	assert.Equal(t, "", userID)

	ctx := req.Context()
	ctx = context.WithValue(ctx, userIDKey, "test_user")
	req = req.WithContext(ctx)

	userID, exists = GetUserID(req)
	assert.True(t, exists)
	assert.Equal(t, "test_user", userID)
}

func TestUserIDKey(t *testing.T) {
	assert.IsType(t, contextKey(""), userIDKey)
	assert.Equal(t, "userID", string(userIDKey))
}
