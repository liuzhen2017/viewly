package handler

import "testing"

func accs(n int) []uint64 {
	out := make([]uint64, n)
	for i := range out {
		out[i] = uint64(100 + i)
	}
	return out
}

func clips(n int) []uint64 {
	out := make([]uint64, n)
	for i := range out {
		out[i] = uint64(200 + i)
	}
	return out
}

type pairCount struct {
	accounts map[uint64]int
	clips    map[uint64]int
	pairs    map[uint64]map[uint64]bool // account -> set of clips
}

func count(pairs []ttAllocPair) pairCount {
	pc := pairCount{accounts: map[uint64]int{}, clips: map[uint64]int{}, pairs: map[uint64]map[uint64]bool{}}
	for _, p := range pairs {
		pc.accounts[p.AccountID]++
		pc.clips[p.ClipID]++
		if pc.pairs[p.AccountID] == nil {
			pc.pairs[p.AccountID] = map[uint64]bool{}
		}
		pc.pairs[p.AccountID][p.ClipID] = true
	}
	return pc
}

func TestAllocateEmpty(t *testing.T) {
	if got := allocate(nil, clips(3), RuleOneToOne, false, 1); got != nil {
		t.Fatalf("no accounts: want nil, got %v", got)
	}
	if got := allocate(accs(3), nil, RuleAllVideos, true, 2); got != nil {
		t.Fatalf("no clips: want nil, got %v", got)
	}
}

func TestAllocateOneToOne(t *testing.T) {
	got := allocate(accs(3), clips(3), RuleOneToOne, false, 1)
	if len(got) != 3 {
		t.Fatalf("want 3 pairs, got %d", len(got))
	}
	for i, p := range got {
		if p.AccountID != uint64(100+i) || p.ClipID != uint64(200+i) {
			t.Fatalf("pair %d: want (%d,%d), got (%d,%d)", i, 100+i, 200+i, p.AccountID, p.ClipID)
		}
	}
}

func TestAllocateOneToOneMoreClipsThanAccounts(t *testing.T) {
	// 2 accounts, 5 clips: extra clips unused — each account one video
	got := allocate(accs(2), clips(5), RuleOneToOne, false, 1)
	if len(got) != 2 {
		t.Fatalf("want 2 pairs, got %d", len(got))
	}
}

func TestAllocateOneToOneLoopWhenShort(t *testing.T) {
	// 4 accounts, 2 clips, loop on: clips cycle so every account posts
	got := allocate(accs(4), clips(2), RuleOneToOne, true, 1)
	if len(got) != 4 {
		t.Fatalf("want 4 pairs, got %d", len(got))
	}
	wantClip := []uint64{200, 201, 200, 201}
	for i, p := range got {
		if p.ClipID != wantClip[i] {
			t.Fatalf("pair %d: want clip %d, got %d", i, wantClip[i], p.ClipID)
		}
	}
}

func TestAllocateOneToOneNoLoopLeavesIdle(t *testing.T) {
	got := allocate(accs(4), clips(2), RuleOneToOne, false, 1)
	if len(got) != 2 {
		t.Fatalf("want 2 pairs (2 accounts idle), got %d", len(got))
	}
}

func TestAllocateAllVideos(t *testing.T) {
	got := allocate(accs(2), clips(3), RuleAllVideos, false, 1)
	if len(got) != 6 {
		t.Fatalf("want 6 pairs, got %d", len(got))
	}
	pc := count(got)
	for _, a := range accs(2) {
		if pc.accounts[a] != 3 {
			t.Fatalf("account %d: want 3 clips, got %d", a, pc.accounts[a])
		}
	}
}

func TestAllocateRandomEvenRespectsCap(t *testing.T) {
	// 10 accounts, 2 clips, cap 2: at most 4 posts, each clip exactly ≤2,
	// no account holds the same clip twice
	got := allocate(accs(10), clips(2), RuleRandomEven, false, 2)
	if len(got) != 4 {
		t.Fatalf("want 4 pairs (pool capped), got %d", len(got))
	}
	pc := count(got)
	for _, cl := range clips(2) {
		if pc.clips[cl] != 2 {
			t.Fatalf("clip %d: want exactly 2 uses, got %d", cl, pc.clips[cl])
		}
	}
	for a, set := range pc.pairs {
		if len(set) != pc.accounts[a] {
			t.Fatalf("account %d got duplicate clips", a)
		}
	}
}

func TestAllocateRandomEvenCapFloor(t *testing.T) {
	// maxRepeat < 1 treated as 1
	got := allocate(accs(3), clips(3), RuleRandomEven, false, 0)
	if len(got) != 3 {
		t.Fatalf("want 3 pairs, got %d", len(got))
	}
	pc := count(got)
	for _, cl := range clips(3) {
		if pc.clips[cl] != 1 {
			t.Fatalf("clip %d: want 1 use, got %d", cl, pc.clips[cl])
		}
	}
}

func TestAllocateUnknownRuleFallsBackToOneToOne(t *testing.T) {
	got := allocate(accs(3), clips(5), "BOGUS", false, 1)
	if len(got) != 3 {
		t.Fatalf("want 3 pairs, got %d", len(got))
	}
}
