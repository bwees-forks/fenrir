package moonlight

import "sync"

// userLocks serializes launch, resume and cancel per user. Moonlight can send
// a second request while the first is still waiting on the game pod, and the
// second one would otherwise delete the session the first is waiting on.
type userLocks struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func (u *userLocks) Lock(user string) (unlock func()) {
	u.mu.Lock()
	if u.locks == nil {
		u.locks = map[string]*sync.Mutex{}
	}
	lock, ok := u.locks[user]
	if !ok {
		lock = &sync.Mutex{}
		u.locks[user] = lock
	}
	u.mu.Unlock()

	lock.Lock()
	return lock.Unlock
}
