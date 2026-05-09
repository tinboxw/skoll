package cache

import (
	"testing"
	"time"
)

func TestNewBundleModes(t *testing.T) {
	cases := []struct {
		name string
		opts Options
	}{
		{name: "local", opts: Options{Mode: ModeLocal, LocalSize: 16}},
		{name: "redis", opts: Options{Mode: ModeRedis, RedisAddr: "127.0.0.1:6379"}},
		{name: "memcached", opts: Options{Mode: ModeMemcached, MemcachedAddr: "127.0.0.1:11211"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := NewBundle(tc.opts)
			if err != nil {
				t.Fatalf("NewBundle error: %v", err)
			}
			if b.Session == nil || b.User == nil || b.Permission == nil || b.Page == nil || b.Config == nil {
				t.Fatalf("bundle has nil cache field")
			}

			b.Session.Set("sid-1", []byte("session-data"), time.Minute)
			if got, ok := b.Session.Get("sid-1"); !ok || string(got) != "session-data" {
				t.Fatalf("session cache contract broken")
			}

			b.Page.Set("/dashboard", []byte("html"), time.Minute)
			if got, ok := b.Page.Get("/dashboard"); !ok || string(got) != "html" {
				t.Fatalf("page cache contract broken")
			}
		})
	}
}

func TestNewBundleErrors(t *testing.T) {
	if _, err := NewBundle(Options{Mode: ModeRedis}); err == nil {
		t.Fatalf("expected redis addr error")
	}
	if _, err := NewBundle(Options{Mode: ModeMemcached}); err == nil {
		t.Fatalf("expected memcached addr error")
	}
	if _, err := NewBundle(Options{Mode: Mode("bad")}); err == nil {
		t.Fatalf("expected unsupported mode error")
	}
}
