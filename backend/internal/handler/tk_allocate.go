package handler

import "math/rand"

// Batch allocation rules for publish tasks — the "matrix engine" copied from
// the reference product. Pure function, no DB: the task creator turns the
// returned pairs into queued tiktok_posts rows.

const (
	RuleOneToOne   = "ONE_TO_ONE"   // each account posts one video (accounts run in parallel)
	RuleAllVideos  = "ALL_VIDEOS"   // every account posts every video (full cross product)
	RuleRandomEven = "RANDOM_EVEN"  // videos spread across accounts, per-video repeat capped
)

type ttAllocPair struct {
	AccountID uint64
	ClipID    uint64
}

// allocate decides who posts what.
//   - ONE_TO_ONE: account i gets clip i; accounts beyond the clip count stay
//     idle unless loopWhenShort cycles clips for them.
//   - ALL_VIDEOS: full cross product (what the admin page always did).
//   - RANDOM_EVEN: a shuffled pool where each clip appears at most maxRepeat
//     times is dealt to accounts in order — even spread, no account gets two
//     copies of the same clip, accounts beyond the pool stay idle.
func allocate(accounts, clips []uint64, rule string, loopWhenShort bool, maxRepeat int) []ttAllocPair {
	if len(accounts) == 0 || len(clips) == 0 {
		return nil
	}
	if maxRepeat < 1 {
		maxRepeat = 1
	}
	var out []ttAllocPair
	switch rule {
	case RuleAllVideos:
		for _, a := range accounts {
			for _, cl := range clips {
				out = append(out, ttAllocPair{AccountID: a, ClipID: cl})
			}
		}
	case RuleRandomEven:
		pool := make([]uint64, 0, len(clips)*maxRepeat)
		for _, cl := range clips {
			for r := 0; r < maxRepeat; r++ {
				pool = append(pool, cl)
			}
		}
		rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
		if len(pool) > len(accounts) {
			pool = pool[:len(accounts)]
		}
		for i, cl := range pool {
			out = append(out, ttAllocPair{AccountID: accounts[i], ClipID: cl})
		}
	default: // ONE_TO_ONE
		for i, a := range accounts {
			if i < len(clips) {
				out = append(out, ttAllocPair{AccountID: a, ClipID: clips[i]})
			} else if loopWhenShort {
				out = append(out, ttAllocPair{AccountID: a, ClipID: clips[i%len(clips)]})
			}
		}
	}
	return out
}
