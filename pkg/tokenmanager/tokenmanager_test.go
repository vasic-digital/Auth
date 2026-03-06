package tokenmanager

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memoryStorage is an in-memory Storage implementation for testing.
type memoryStorage struct {
	data      map[string]string
	failStore bool
}

func newMemoryStorage() *memoryStorage {
	return &memoryStorage{data: make(map[string]string)}
}

func (m *memoryStorage) Store(key, value string) error {
	if m.failStore {
		return assert.AnError
	}
	m.data[key] = value
	return nil
}

func (m *memoryStorage) Retrieve(key string) (string, error) {
	return m.data[key], nil
}

func (m *memoryStorage) Delete(key string) error {
	delete(m.data, key)
	return nil
}

func TestNew(t *testing.T) {
	storage := newMemoryStorage()
	mgr := New("test-service", storage)
	assert.Equal(t, "test-service", mgr.ServiceName())
}

func TestStoreAndGetAccessToken(t *testing.T) {
	storage := newMemoryStorage()
	mgr := New("svc", storage)

	err := mgr.StoreAccessToken("my-access-token")
	require.NoError(t, err)

	token, err := mgr.GetAccessToken()
	require.NoError(t, err)
	assert.Equal(t, "my-access-token", token)
}

func TestGetAccessTokenNotFound(t *testing.T) {
	storage := newMemoryStorage()
	mgr := New("svc", storage)

	token, err := mgr.GetAccessToken()
	require.NoError(t, err)
	assert.Empty(t, token)
}

func TestStoreAndGetRefreshToken(t *testing.T) {
	storage := newMemoryStorage()
	mgr := New("svc", storage)

	err := mgr.StoreRefreshToken("my-refresh-token")
	require.NoError(t, err)

	token, err := mgr.GetRefreshToken()
	require.NoError(t, err)
	assert.Equal(t, "my-refresh-token", token)
}

func TestIsExpired_ValidToken(t *testing.T) {
	storage := newMemoryStorage()
	mgr := New("svc", storage)

	err := mgr.StoreExpiration(time.Now().Add(1 * time.Hour))
	require.NoError(t, err)

	expired, err := mgr.IsExpired()
	require.NoError(t, err)
	assert.False(t, expired)
}

func TestIsExpired_ExpiredToken(t *testing.T) {
	storage := newMemoryStorage()
	mgr := New("svc", storage)

	err := mgr.StoreExpiration(time.Now().Add(-1 * time.Hour))
	require.NoError(t, err)

	expired, err := mgr.IsExpired()
	require.NoError(t, err)
	assert.True(t, expired)
}

func TestIsExpired_NoExpiration(t *testing.T) {
	storage := newMemoryStorage()
	mgr := New("svc", storage)

	expired, err := mgr.IsExpired()
	require.NoError(t, err)
	assert.True(t, expired, "no expiration stored should be treated as expired")
}

func TestHasValidToken_Valid(t *testing.T) {
	storage := newMemoryStorage()
	mgr := New("svc", storage)

	require.NoError(t, mgr.StoreAccessToken("token"))
	require.NoError(t, mgr.StoreExpiration(time.Now().Add(1*time.Hour)))

	valid, err := mgr.HasValidToken()
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestHasValidToken_Expired(t *testing.T) {
	storage := newMemoryStorage()
	mgr := New("svc", storage)

	require.NoError(t, mgr.StoreAccessToken("token"))
	require.NoError(t, mgr.StoreExpiration(time.Now().Add(-1*time.Hour)))

	valid, err := mgr.HasValidToken()
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestHasValidToken_NoToken(t *testing.T) {
	storage := newMemoryStorage()
	mgr := New("svc", storage)

	valid, err := mgr.HasValidToken()
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestClearTokens(t *testing.T) {
	storage := newMemoryStorage()
	mgr := New("svc", storage)

	require.NoError(t, mgr.StoreAccessToken("token"))
	require.NoError(t, mgr.StoreRefreshToken("refresh"))
	require.NoError(t, mgr.StoreExpiration(time.Now().Add(1*time.Hour)))

	err := mgr.ClearTokens()
	require.NoError(t, err)

	valid, err := mgr.HasValidToken()
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestStoreTokenInfo(t *testing.T) {
	storage := newMemoryStorage()
	mgr := New("svc", storage)

	err := mgr.StoreTokenInfo("access", "refresh", 1*time.Hour)
	require.NoError(t, err)

	info, err := mgr.GetTokenInfo()
	require.NoError(t, err)
	assert.True(t, info.HasAccessToken)
	assert.True(t, info.HasRefreshToken)
	assert.False(t, info.IsExpired)
	assert.Equal(t, "svc", info.ServiceName)
}

func TestGetTokenInfo_NoTokens(t *testing.T) {
	storage := newMemoryStorage()
	mgr := New("svc", storage)

	info, err := mgr.GetTokenInfo()
	require.NoError(t, err)
	assert.False(t, info.HasAccessToken)
	assert.False(t, info.HasRefreshToken)
	assert.True(t, info.IsExpired)
}

func TestStorageFailure(t *testing.T) {
	storage := newMemoryStorage()
	storage.failStore = true
	mgr := New("svc", storage)

	err := mgr.StoreAccessToken("token")
	assert.Error(t, err)
}

func TestMultipleServices(t *testing.T) {
	storage := newMemoryStorage()
	mgr1 := New("svc1", storage)
	mgr2 := New("svc2", storage)

	require.NoError(t, mgr1.StoreAccessToken("token1"))
	require.NoError(t, mgr2.StoreAccessToken("token2"))

	t1, _ := mgr1.GetAccessToken()
	t2, _ := mgr2.GetAccessToken()
	assert.Equal(t, "token1", t1)
	assert.Equal(t, "token2", t2)
}
