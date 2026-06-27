package limiter

import "time"

type Account struct {
	ID            uint
	Cookie        string
	RefreshToken  string
	FailedCount   uint
	NextAvailable time.Time
	LastRequest   time.Time
	refreshFunc   func(id uint) // called on consecutive failures
}

func (a *Account) Available() bool {
	return a.NextAvailable.Before(time.Now())
}

func (a *Account) Penalize() {
	a.FailedCount++
	if a.FailedCount >= 3 && a.refreshFunc != nil {
		a.refreshFunc(a.ID)
		a.FailedCount = 0
		return
	}
	a.NextAvailable = time.Now().Add(5 * time.Minute)
}

func (a *Account) Reward() {
	a.FailedCount = 0
	a.NextAvailable = time.Time{}
}
