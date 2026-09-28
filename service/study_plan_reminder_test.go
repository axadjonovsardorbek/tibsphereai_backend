package service

import "testing"

func TestParseHHMMToMinutes(t *testing.T) {
	cases := []struct {
		in     string
		want   int
		wantOk bool
	}{
		{"20:00:00", 20 * 60, true},
		{"20:00", 20 * 60, true},
		{"00:05", 5, true},
		{"23:59:00", 23*60 + 59, true},
		{"not-a-time", 0, false},
		{"", 0, false},
	}
	for _, c := range cases {
		got, ok := parseHHMMToMinutes(c.in)
		if ok != c.wantOk {
			t.Fatalf("parseHHMMToMinutes(%q) ok = %v, want %v", c.in, ok, c.wantOk)
		}
		if ok && got != c.want {
			t.Fatalf("parseHHMMToMinutes(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestReminderDueNow(t *testing.T) {
	cases := []struct {
		name          string
		remindMinutes int
		before        int
		nowMinutes    int
		want          bool
	}{
		{"exact match, no offset", 20 * 60, 0, 20 * 60, true},
		{"15 min before, matches", 20 * 60, 15, 20*60 - 15, true},
		{"15 min before, not yet", 20 * 60, 15, 20 * 60, false},
		{"wraps past midnight", 5, 15, 24*60 - 10, true}, // 00:05 - 15m = 23:50
		{"no match", 20 * 60, 0, 21 * 60, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := reminderDueNow(c.remindMinutes, c.before, c.nowMinutes)
			if got != c.want {
				t.Fatalf("reminderDueNow(%d,%d,%d) = %v, want %v", c.remindMinutes, c.before, c.nowMinutes, got, c.want)
			}
		})
	}
}
