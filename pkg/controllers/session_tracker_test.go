package controllers

import (
	"fmt"
	"sync"
	"testing"
)

func TestSessionTrackerUntrackReportsLastSession(t *testing.T) {
	tracker := newSessionTracker()
	ug := userGame{User: "alice", Game: "steam"}
	tracker.Track("old", ug)
	tracker.Track("new", ug)

	got, last, found := tracker.Untrack("old")
	if !found || got != ug || last {
		t.Fatalf("Untrack(old) = %v, %v, %v; want %v, false, true", got, last, found, ug)
	}

	got, last, found = tracker.Untrack("new")
	if !found || got != ug || !last {
		t.Fatalf("Untrack(new) = %v, %v, %v; want %v, true, true", got, last, found, ug)
	}

	if _, _, found := tracker.Untrack("new"); found {
		t.Fatal("Untrack of an unknown session reported found")
	}
	if names := tracker.Sessions(ug); len(names) != 0 {
		t.Fatalf("Sessions after untracking all = %v, want none", names)
	}
}

func TestSessionTrackerConcurrentAccess(t *testing.T) {
	tracker := newSessionTracker()
	ug := userGame{User: "alice", Game: "steam"}

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() {
			name := fmt.Sprintf("session-%d", i)
			tracker.Track(name, ug)
			tracker.Sessions(ug)
			tracker.Untrack(name)
		})
	}
	wg.Wait()

	if names := tracker.Sessions(ug); len(names) != 0 {
		t.Fatalf("Sessions = %v, want none", names)
	}
}

func TestUserGameDeploymentName(t *testing.T) {
	if got := (userGame{User: "alice", Game: "steam"}).deploymentName(); got != "alice-steam" {
		t.Fatalf("deploymentName = %q, want alice-steam", got)
	}
}
