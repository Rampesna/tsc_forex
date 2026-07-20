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

func TestEveryBadgeHasDeterministicProgress(t *testing.T) {
	if len(badgeCatalog) != 30 {
		t.Fatalf("badge catalog contains %d badges, want exactly 30", len(badgeCatalog))
	}
	st := &models.UserGamification{Level: 7, LongestStreak: 8, TotalCheckIns: 31}
	for _, badge := range badgeCatalog {
		current, target := badgeProgress(badge.Code, st, 5, 2)
		if target <= 0 {
			t.Errorf("badgeProgress(%q) target = %d, want positive target", badge.Code, target)
		}
		if current < 0 {
			t.Errorf("badgeProgress(%q) current = %d, want non-negative current", badge.Code, current)
		}
		if got := badgeQualifies(badge.Code, st, 5, 2); got != (current >= target) {
			t.Errorf("badgeQualifies(%q) = %v, progress says %d/%d", badge.Code, got, current, target)
		}
	}
}

func TestBadgeStatusClampsProgress(t *testing.T) {
	st := models.UserGamification{TotalCheckIns: 50}
	status := badgeStatus(badgeCatalog[0], st, 0, 0, true, nil)
	if status.Current != 50 || status.Target != 1 || status.Progress != 1 {
		t.Fatalf("badge status progress = %d/%d (%.2f), want 50/1 (1.00)",
			status.Current, status.Target, status.Progress)
	}

	status = badgeStatus(badgeCatalog[12], models.UserGamification{}, 5, 0, false, nil)
	if status.Code != "collector_25" || status.Current != 5 || status.Target != 25 || status.Progress != 0.2 {
		t.Fatalf("market radar progress = %#v, want 5/25 (0.20)", status)
	}

	status = badgeStatus(badgeCatalog[18], models.UserGamification{}, 0, 0, true, nil)
	if status.Code != "watchful_50" || status.Current != 50 || status.Target != 50 || status.Progress != 1 {
		t.Fatalf("earned badge must stay complete when a counter read is unavailable: %#v", status)
	}
}
