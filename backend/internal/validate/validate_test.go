package validate

import "testing"

func TestMoney(t *testing.T) {
	cases := []struct {
		in    string
		paise int64
		ok    bool
	}{
		{"5000.00", 500000, true},
		{"5000", 500000, true},
		{"0.5", 50, true},
		{"1.234", 0, false}, // 3 decimals
		{"-5", 0, false},
		{"1e3", 0, false},
		{" 5", 0, false},
		{"01", 0, false},
		{"", 0, false},
	}
	for _, tc := range cases {
		got, ok := Money(tc.in)
		if ok != tc.ok || got != tc.paise {
			t.Errorf("Money(%q) = %d,%v want %d,%v", tc.in, got, ok, tc.paise, tc.ok)
		}
	}
}

func TestText(t *testing.T) {
	cases := map[string]string{
		"  Mom  ":  "Mom",
		"a\n\tb":   "a b",
		"a\u200bb": "ab", // zero-width space
		"x\u202ey": "xy", // RTL override (spoofing trick)
	}
	for in, want := range cases {
		if got := Text(in); got != want {
			t.Errorf("Text(%q) = %q want %q", in, got, want)
		}
	}
}
