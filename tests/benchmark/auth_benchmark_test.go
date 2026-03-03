package benchmark

import (
	"testing"
	"time"

	"digital.vasic.auth/pkg/apikey"
	"digital.vasic.auth/pkg/jwt"
	"digital.vasic.auth/pkg/token"
)

func BenchmarkJWTCreate(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	mgr := jwt.NewManager(jwt.DefaultConfig("bench-secret-key"))
	claims := map[string]interface{}{
		"sub":  "user-bench",
		"role": "admin",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mgr.Create(claims)
	}
}

func BenchmarkJWTValidate(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	mgr := jwt.NewManager(jwt.DefaultConfig("bench-secret-key"))
	tokenStr, _ := mgr.Create(map[string]interface{}{
		"sub": "user-bench",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mgr.Validate(tokenStr)
	}
}

func BenchmarkJWTRefresh(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	mgr := jwt.NewManager(jwt.DefaultConfig("bench-secret-key"))
	tokenStr, _ := mgr.Create(map[string]interface{}{
		"sub": "user-bench",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mgr.Refresh(tokenStr)
	}
}

func BenchmarkAPIKeyGenerate(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	gen := apikey.NewGenerator(apikey.DefaultGeneratorConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = gen.Generate("bench", []string{"read"}, time.Time{})
	}
}

func BenchmarkAPIKeyStoreGet(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	store := apikey.NewInMemoryStore()
	gen := apikey.NewGenerator(apikey.DefaultGeneratorConfig())
	key, _ := gen.Generate("bench", []string{"read"}, time.Time{})
	_ = store.Store(key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.Get(key.Key)
	}
}

func BenchmarkTokenStoreSetGet(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	store := token.NewInMemoryStore()
	tok := token.NewSimpleToken("access", "refresh", time.Now().Add(1*time.Hour))
	_ = store.Set("bench-key", tok, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.Set("bench-key", tok, 0)
		_, _ = store.Get("bench-key")
	}
}

func BenchmarkAPIKeyMaskKey(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	key := "ak-1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = apikey.MaskKey(key)
	}
}

func BenchmarkTokenStoreCleanup(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark test in short mode")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		store := token.NewInMemoryStore()
		for j := 0; j < 100; j++ {
			tok := token.NewSimpleToken("a", "r", time.Now().Add(1*time.Hour))
			_ = store.Set(string(rune(j)), tok, time.Nanosecond)
		}
		time.Sleep(2 * time.Millisecond)
		b.StartTimer()
		store.Cleanup()
	}
}
