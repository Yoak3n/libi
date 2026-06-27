package limiter

import (
	"context"
	"maps"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type AccountLimiter struct {
	limiters     map[uint]*rate.Limiter
	accounts     map[uint]Account
	mu           sync.Mutex
	refreshFunc  func(id uint, cookie string) string
	interval     time.Duration
}

func NewAccountLimiter(interval time.Duration) *AccountLimiter {
	if interval <= 0 {
		interval = 3 * time.Second
	}
	return &AccountLimiter{
		limiters: make(map[uint]*rate.Limiter),
		accounts: make(map[uint]Account),
		interval: interval,
	}
}

// SetRefreshFunc sets the callback that refreshes a cookie.
// It receives the account ID and old cookie, returns the new cookie (or empty if refresh failed).
func (a *AccountLimiter) SetRefreshFunc(fn func(id uint, cookie string) string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.refreshFunc = fn
}

func (a *AccountLimiter) SetAccount(id uint, cookie, refreshToken string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.limiters[id] = rate.NewLimiter(rate.Every(2*time.Second), 4)
	a.accounts[id] = Account{
		ID:           id,
		Cookie:       cookie,
		RefreshToken: refreshToken,
		refreshFunc:  a.makeRefreshFunc(id),
	}
}

func (a *AccountLimiter) makeRefreshFunc(id uint) func(uint) {
	return func(failedID uint) {
		a.mu.Lock()
		account := a.accounts[failedID]
		oldCookie := account.Cookie
		a.mu.Unlock()

		if a.refreshFunc == nil {
			return
		}
		newCookie := a.refreshFunc(failedID, oldCookie)
		if newCookie == "" {
			return
		}

		a.mu.Lock()
		defer a.mu.Unlock()
		account = a.accounts[failedID]
		account.Cookie = newCookie
		account.FailedCount = 0
		account.NextAvailable = time.Time{}
		a.accounts[failedID] = account
	}
}

func (a *AccountLimiter) Wait(ctx context.Context, id uint) error {
	a.mu.Lock()
	lim := a.limiters[id]
	a.mu.Unlock()
	if lim == nil {
		return nil
	}
	return lim.Wait(ctx)
}

func (a *AccountLimiter) Snapshot() map[uint]Account {
	a.mu.Lock()
	defer a.mu.Unlock()
	snapshot := make(map[uint]Account)
	maps.Copy(snapshot, a.accounts)
	return snapshot
}

func (a *AccountLimiter) GetAccount(ctx context.Context) (uint, string) {
	snapshot := a.Snapshot()
	for k, v := range snapshot {
		if v.Available() {
			a.Wait(ctx, k)
			return k, v.Cookie
		}
	}
	return 0, ""
}

// PickAccount returns the best available account and how long to wait before
// making the request. Returns (0, "", 0) if no account exists at all.
// The caller must sleep for waitTime before issuing the request.
func (a *AccountLimiter) PickAccount() (id uint, cookie string, waitTime time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()

	var bestID uint
	var bestWait time.Duration
	first := true

	for k, v := range a.accounts {
		if !v.Available() {
			continue
		}
		elapsed := time.Since(v.LastRequest)
		needWait := a.interval - elapsed
		if needWait < 0 {
			needWait = 0
		}
		if first || needWait < bestWait {
			bestID = k
			bestWait = needWait
			first = false
		}
	}

	if bestID == 0 {
		// No available account — find soonest to recover
		var soonest time.Time
		for k, v := range a.accounts {
			if soonest.IsZero() || v.NextAvailable.Before(soonest) {
				soonest = v.NextAvailable
				bestID = k
			}
		}
		if bestID == 0 {
			return 0, "", 0
		}
		wait := time.Until(soonest)
		if wait < 0 {
			wait = 0
		}
		return bestID, a.accounts[bestID].Cookie, wait
	}

	// Update last request time
	acct := a.accounts[bestID]
	acct.LastRequest = time.Now()
	a.accounts[bestID] = acct

	return bestID, a.accounts[bestID].Cookie, bestWait
}

func (a *AccountLimiter) Penalize(id uint) {
	a.mu.Lock()
	defer a.mu.Unlock()
	account := a.accounts[id]
	account.Penalize()
	a.accounts[id] = account
}

func (a *AccountLimiter) Reward(id uint) {
	a.mu.Lock()
	defer a.mu.Unlock()
	account := a.accounts[id]
	account.Reward()
	a.accounts[id] = account
}

// SetInterval updates the request interval at runtime.
func (a *AccountLimiter) SetInterval(d time.Duration) {
	if d > 0 {
		a.mu.Lock()
		a.interval = d
		a.mu.Unlock()
	}
}

// UpdateCookie updates the cookie for an account (used after external refresh).
func (a *AccountLimiter) UpdateCookie(id uint, cookie string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	account := a.accounts[id]
	account.Cookie = cookie
	account.FailedCount = 0
	account.NextAvailable = time.Time{}
	a.accounts[id] = account
}
