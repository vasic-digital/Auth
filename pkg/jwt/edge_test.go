package jwt_test

import (
	"strings"
	"testing"
	"time"

	"digital.vasic.auth/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_Create_EmptySecret(t *testing.T) {
	t.Parallel()
	cfg := jwt.DefaultConfig("")
	m := jwt.NewManager(cfg)

	token, err := m.Create(map[string]interface{}{"sub": "user"})
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestManager_Validate_ExtremelyLongToken(t *testing.T) {
	t.Parallel()
	m := jwt.NewManager(jwt.DefaultConfig("secret"))

	longToken := strings.Repeat("a", 100000)
	_, err := m.Validate(longToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestManager_Validate_MalformedJWTStrings(t *testing.T) {
	t.Parallel()
	m := jwt.NewManager(jwt.DefaultConfig("secret"))

	tests := []struct {
		name  string
		token string
	}{
		{"empty string", ""},
		{"single dot", "a.b"},
		{"three dots no content", ".."},
		{"three parts garbage", "aaa.bbb.ccc"},
		{"valid base64 but not JWT", "eyJ0eXAi.eyJzdWIi.c2lnbmF0dXJl"},
		{"null bytes", "\x00\x00\x00"},
		{"unicode garbage", "\u0000\u200b\ufeff"},
		{"just whitespace", "   \t\n  "},
		{"SQL injection attempt", "'; DROP TABLE tokens; --"},
		{"HTML injection", "<script>alert('xss')</script>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := m.Validate(tt.token)
			assert.Error(t, err)
		})
	}
}

func TestManager_Validate_ExpiredToken(t *testing.T) {
	t.Parallel()
	cfg := jwt.DefaultConfig("secret")
	cfg.Expiration = -time.Hour
	m := jwt.NewManager(cfg)

	token, err := m.Create(map[string]interface{}{"sub": "user"})
	require.NoError(t, err)

	_, err = m.Validate(token)
	assert.Error(t, err)
}

func TestManager_Create_UnicodeUsernames(t *testing.T) {
	t.Parallel()
	m := jwt.NewManager(jwt.DefaultConfig("secret"))

	tests := []struct {
		name     string
		username string
	}{
		{"chinese", "\u4e2d\u6587\u7528\u6237"},
		{"arabic", "\u0645\u0633\u062a\u062e\u062f\u0645"},
		{"emoji", "\U0001f600\U0001f680\U0001f30d"},
		{"japanese", "\u30e6\u30fc\u30b6\u30fc"},
		{"cyrillic", "\u041f\u043e\u043b\u044c\u0437\u043e\u0432\u0430\u0442\u0435\u043b\u044c"},
		{"mixed scripts", "user_\u4e2d\u6587_\U0001f600"},
		{"zero width joiner", "a\u200db"},
		{"combining marks", "e\u0301\u0327"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			claims := map[string]interface{}{"sub": tt.username}
			token, err := m.Create(claims)
			require.NoError(t, err)

			parsed, err := m.Validate(token)
			require.NoError(t, err)
			assert.Equal(t, tt.username, parsed.Claims["sub"])
		})
	}
}

func TestManager_Create_NilClaims(t *testing.T) {
	t.Parallel()
	m := jwt.NewManager(jwt.DefaultConfig("secret"))

	token, err := m.Create(nil)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	parsed, err := m.Validate(token)
	require.NoError(t, err)
	assert.NotNil(t, parsed.Claims["exp"])
	assert.NotNil(t, parsed.Claims["iat"])
}

func TestManager_Refresh_ExpiredToken(t *testing.T) {
	t.Parallel()
	cfg := jwt.DefaultConfig("secret")
	cfg.Expiration = -time.Hour
	m := jwt.NewManager(cfg)

	token, err := m.Create(map[string]interface{}{"sub": "user"})
	require.NoError(t, err)

	_, err = m.Refresh(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot refresh invalid token")
}

func TestManager_Create_VeryShortExpiration(t *testing.T) {
	t.Parallel()
	cfg := jwt.DefaultConfig("secret")
	cfg.Expiration = time.Nanosecond
	m := jwt.NewManager(cfg)

	token, err := m.Create(map[string]interface{}{"sub": "user"})
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Token may already be expired by the time we validate
	time.Sleep(time.Millisecond)
	_, err = m.Validate(token)
	assert.Error(t, err)
}

func TestManager_Create_MaxExpiration(t *testing.T) {
	t.Parallel()
	cfg := jwt.DefaultConfig("secret")
	cfg.Expiration = 100 * 365 * 24 * time.Hour // 100 years
	m := jwt.NewManager(cfg)

	token, err := m.Create(map[string]interface{}{"sub": "user"})
	require.NoError(t, err)

	parsed, err := m.Validate(token)
	require.NoError(t, err)
	assert.True(t, parsed.ExpiresAt.After(time.Now().Add(99*365*24*time.Hour)))
}

func TestManager_Create_SpecialClaimValues(t *testing.T) {
	t.Parallel()
	m := jwt.NewManager(jwt.DefaultConfig("secret"))

	tests := []struct {
		name   string
		claims map[string]interface{}
	}{
		{"empty string values", map[string]interface{}{"sub": "", "role": ""}},
		{"numeric values", map[string]interface{}{"id": 999999999, "score": 3.14}},
		{"boolean values", map[string]interface{}{"active": true, "verified": false}},
		{"nested map", map[string]interface{}{"data": map[string]interface{}{"key": "value"}}},
		{"slice value", map[string]interface{}{"roles": []string{"admin", "user"}}},
		{"nil value", map[string]interface{}{"data": nil}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			token, err := m.Create(tt.claims)
			require.NoError(t, err)
			assert.NotEmpty(t, token)
		})
	}
}

func TestManager_Validate_WrongSecretToken(t *testing.T) {
	t.Parallel()
	m1 := jwt.NewManager(jwt.DefaultConfig("secret-1"))
	m2 := jwt.NewManager(jwt.DefaultConfig("secret-2"))

	token, err := m1.Create(map[string]interface{}{"sub": "user"})
	require.NoError(t, err)

	_, err = m2.Validate(token)
	assert.Error(t, err)
}

func TestManager_Create_OverrideStandardClaims(t *testing.T) {
	t.Parallel()
	m := jwt.NewManager(jwt.DefaultConfig("secret"))

	// User tries to override exp/iat -- Create should overwrite them
	claims := map[string]interface{}{
		"sub": "user",
		"exp": 0,
		"iat": 0,
	}

	token, err := m.Create(claims)
	require.NoError(t, err)

	parsed, err := m.Validate(token)
	require.NoError(t, err)
	// The token should be valid (not expired) because Create overwrites exp
	assert.True(t, parsed.ExpiresAt.After(time.Now()))
}

func TestManager_Validate_TokenWithExtraWhitespace(t *testing.T) {
	t.Parallel()
	m := jwt.NewManager(jwt.DefaultConfig("secret"))

	token, err := m.Create(map[string]interface{}{"sub": "user"})
	require.NoError(t, err)

	// Tokens with leading/trailing whitespace should fail
	_, err = m.Validate(" " + token)
	assert.Error(t, err)

	_, err = m.Validate(token + " ")
	assert.Error(t, err)

	_, err = m.Validate("\n" + token)
	assert.Error(t, err)
}
