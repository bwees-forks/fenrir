package controllers

import (
	"fmt"
	"slices"
	"sync"

	"k8s.io/apimachinery/pkg/util/sets"
)

type userGame struct {
	User string
	Game string
}

func (ug userGame) deploymentName() string {
	return fmt.Sprintf("%s-%s", ug.User, ug.Game)
}

// sessionTracker maps Session names to the user's game they stream. It is
// shared by the controller's workers, so every access takes the lock.
type sessionTracker struct {
	mu       sync.Mutex
	sessions map[userGame]sets.Set[string]
	games    map[string]userGame
}

func newSessionTracker() *sessionTracker {
	return &sessionTracker{
		sessions: make(map[userGame]sets.Set[string]),
		games:    make(map[string]userGame),
	}
}

func (t *sessionTracker) Track(name string, ug userGame) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if existing, ok := t.sessions[ug]; ok {
		existing.Insert(name)
	} else {
		t.sessions[ug] = sets.New(name)
	}
	t.games[name] = ug
}

// Untrack forgets a session. lastSession is true when no other session of the
// same game is still tracked.
func (t *sessionTracker) Untrack(name string) (ug userGame, lastSession bool, found bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	ug, found = t.games[name]
	if !found {
		return userGame{}, false, false
	}
	delete(t.games, name)

	existing := t.sessions[ug]
	existing.Delete(name)
	if existing.Len() > 0 {
		return ug, false, true
	}
	delete(t.sessions, ug)
	return ug, true, true
}

func (t *sessionTracker) Sessions(ug userGame) []string {
	t.mu.Lock()
	defer t.mu.Unlock()

	names := t.sessions[ug].UnsortedList()
	slices.Sort(names)
	return names
}
