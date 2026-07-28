package session

import "testing"

func TestFakeAuthStateUsesLoopbackSnapshotPlatform(t *testing.T) {
	t.Setenv("CRATER_TEST_SANDBOX_PLATFORM_URL", "http://127.0.0.1:38080/")

	state := fakeAuthState()
	if state.ActiveContext.PlatformURL != "http://127.0.0.1:38080" {
		t.Fatalf("unexpected active platform URL: %q", state.ActiveContext.PlatformURL)
	}
	for _, info := range state.AuthInfos[:2] {
		if info.PlatformURL != state.ActiveContext.PlatformURL {
			t.Fatalf("auth info platform URL = %q, want %q", info.PlatformURL, state.ActiveContext.PlatformURL)
		}
	}
}

func TestFakeAuthStateRejectsNonLoopbackSnapshotPlatform(t *testing.T) {
	t.Setenv("CRATER_TEST_SANDBOX_PLATFORM_URL", "https://example.com")

	state := fakeAuthState()
	if state.ActiveContext.PlatformURL != "https://example.invalid" {
		t.Fatalf("unexpected active platform URL: %q", state.ActiveContext.PlatformURL)
	}
}
