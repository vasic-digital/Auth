package gin_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"digital.vasic.auth/pkg/accesstoken"
	adapter "digital.vasic.auth/pkg/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAccessTokenAuth_ValidToken(t *testing.T) {
	store := accesstoken.NewMemoryStore()
	tok, _ := accesstoken.Generate(32)
	store.Store(tok, &accesstoken.TokenInfo{
		UserID:    "user-42",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})

	r := gin.New()
	r.Use(adapter.AccessTokenAuth(store, adapter.DefaultConfig()))
	r.GET("/test", func(c *gin.Context) {
		userID, _ := c.Get(adapter.CtxUserID)
		c.String(http.StatusOK, userID.(string))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Access-Token", tok)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "user-42", w.Body.String())
}

func TestAccessTokenAuth_TokenFromQuery(t *testing.T) {
	store := accesstoken.NewMemoryStore()
	tok, _ := accesstoken.Generate(32)
	store.Store(tok, &accesstoken.TokenInfo{
		UserID:    "user-99",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})

	r := gin.New()
	r.Use(adapter.AccessTokenAuth(store, adapter.DefaultConfig()))
	r.GET("/test", func(c *gin.Context) {
		userID, _ := c.Get(adapter.CtxUserID)
		c.String(http.StatusOK, userID.(string))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test?access="+tok, nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "user-99", w.Body.String())
}

func TestAccessTokenAuth_MissingToken(t *testing.T) {
	store := accesstoken.NewMemoryStore()

	r := gin.New()
	r.Use(adapter.AccessTokenAuth(store, adapter.DefaultConfig()))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "should not reach")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAccessTokenAuth_InvalidToken(t *testing.T) {
	store := accesstoken.NewMemoryStore()

	r := gin.New()
	r.Use(adapter.AccessTokenAuth(store, adapter.DefaultConfig()))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "should not reach")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Access-Token", "bogus-token")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOptionalAuth_WithToken(t *testing.T) {
	store := accesstoken.NewMemoryStore()
	tok, _ := accesstoken.Generate(32)
	store.Store(tok, &accesstoken.TokenInfo{
		UserID:    "user-1",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})

	r := gin.New()
	r.Use(adapter.OptionalAuth(store, adapter.DefaultConfig()))
	r.GET("/test", func(c *gin.Context) {
		userID, exists := c.Get(adapter.CtxUserID)
		if exists {
			c.String(http.StatusOK, userID.(string))
		} else {
			c.String(http.StatusOK, "anonymous")
		}
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Access-Token", tok)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "user-1", w.Body.String())
}

func TestOptionalAuth_WithoutToken(t *testing.T) {
	store := accesstoken.NewMemoryStore()

	r := gin.New()
	r.Use(adapter.OptionalAuth(store, adapter.DefaultConfig()))
	r.GET("/test", func(c *gin.Context) {
		_, exists := c.Get(adapter.CtxUserID)
		if exists {
			c.String(http.StatusOK, "authenticated")
		} else {
			c.String(http.StatusOK, "anonymous")
		}
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "anonymous", w.Body.String())
}
