package web

import "testing"

func TestResolveAddr(t *testing.T) {
	t.Setenv("PORT", "")
	got, err := ResolveAddr("")
	if err != nil {
		t.Fatal(err)
	}
	if got != DefaultAddr {
		t.Fatalf("got %s", got)
	}
	got, err = ResolveAddr("127.0.0.1:19111")
	if err != nil || got != "127.0.0.1:19111" {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err = ResolveAddr("0.0.0.0:19111"); err == nil {
		t.Fatal("应拒绝非回环地址")
	}
}

func TestResolveAddrFromPort(t *testing.T) {
	t.Setenv("PORT", "19112")
	got, err := ResolveAddr("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "127.0.0.1:19112" {
		t.Fatalf("got %s", got)
	}
}
