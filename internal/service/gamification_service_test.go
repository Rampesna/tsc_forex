package service

import (
	"testing"

	"github.com/tsc-forex/tsc_forex/internal/models"
)

func TestLevelForXP(t *testing.T) {
	cases := []struct {
		xp    int64
		level int
	}{
		{0, 1},
		{49, 1},
		{50, 2},
		{199, 2},
		{200, 3},
		{800, 5},
		{4050, 10},
		{28800, 25},
	}
	for _, c := range cases {
		if got := levelForXP(c.xp); got != c.level {
			t.Errorf("levelForXP(%d) = %d, want %d", c.xp, got, c.level)
		}
	}
}

func TestXPForLevelRoundTrip(t *testing.T) {
	// The XP required to reach level L must itself resolve back to level L.
	for l := 1; l <= 60; l++ {
		xp := xpForLevel(l)
		if got := levelForXP(xp); got != l {
			t.Errorf("xpForLevel(%d)=%d -> levelForXP=%d, want %d", l, xp, got, l)
		}
		// One XP below the threshold must be the previous level.
		if l > 1 {
			if got := levelForXP(xp - 1); got != l-1 {
				t.Errorf("levelForXP(%d) = %d, want %d", xp-1, got, l-1)
			}
		}
	}
}

func TestBadgeQualifies(t *testing.T) {
	st := &models.UserGamification{Level: 7, LongestStreak: 8, TotalCheckIns: 31}
	want := map[string]bool{
		"first_steps": true,
		"streak_3":    true,
		"streak_7":    true,
		"streak_30":   false,
		"level_5":     true,
		"level_10":    false,
		"committed":   true,
		"collector":   false,
		"watchful":    true,
	}
	for code, exp := range want {
		favCount, alertCount := int64(2), int64(6)
		if got := badgeQualifies(code, st, favCount, alertCount); got != exp {
			t.Errorf("badgeQualifies(%q) = %v, want %v", code, got, exp)
		}
	}
}
