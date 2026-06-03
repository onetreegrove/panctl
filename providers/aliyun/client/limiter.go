package client

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/time/rate"
)

type limiterType int

const (
	limiterList limiterType = iota
	limiterLink
	limiterOther
)

const (
	defaultListRate  = 3.9
	defaultLinkRate  = 0.9
	defaultOtherRate = 14.9
	globalLimiterKey = ""
)

type userLimiter struct {
	usedBy int
	list   *rate.Limiter
	link   *rate.Limiter
	other  *rate.Limiter
}

var (
	limitersMu sync.Mutex
	limiters   = map[string]*userLimiter{}
)

func newUserLimiter(baseRate float64) *userLimiter {
	if baseRate <= 0 {
		baseRate = 1
	}
	return &userLimiter{
		usedBy: 1,
		list:   rate.NewLimiter(rate.Limit(baseRate), 1),
		link:   rate.NewLimiter(rate.Limit(baseRate), 1),
		other:  rate.NewLimiter(rate.Limit(baseRate), 1),
	}
}

func newAliyunLimiter() *userLimiter {
	return &userLimiter{
		usedBy: 1,
		list:   rate.NewLimiter(rate.Limit(defaultListRate), 1),
		link:   rate.NewLimiter(rate.Limit(defaultLinkRate), 1),
		other:  rate.NewLimiter(rate.Limit(defaultOtherRate), 1),
	}
}

func limiterForUser(userID string) *userLimiter {
	limitersMu.Lock()
	defer limitersMu.Unlock()
	if lim, ok := limiters[userID]; ok {
		lim.usedBy++
		return lim
	}
	lim := newAliyunLimiter()
	limiters[userID] = lim
	return lim
}

func (l *userLimiter) wait(ctx context.Context, typ limiterType) error {
	if l == nil {
		return fmt.Errorf("aliyun client not initialized")
	}
	switch typ {
	case limiterList:
		return l.list.Wait(ctx)
	case limiterLink:
		return l.link.Wait(ctx)
	case limiterOther:
		return l.other.Wait(ctx)
	default:
		return fmt.Errorf("unknown aliyun limiter type")
	}
}
