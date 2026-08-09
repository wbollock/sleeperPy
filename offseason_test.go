package main

import "testing"

func TestIsOffseasonMode(t *testing.T) {
	cases := []struct {
		name     string
		state    map[string]interface{}
		expected bool
	}{
		{
			name:     "true offseason",
			state:    map[string]interface{}{"season_type": "off", "week": 1.0},
			expected: true,
		},
		{
			name:     "preseason week 1 -- still offseason mode",
			state:    map[string]interface{}{"season_type": "pre", "week": 1.0},
			expected: true,
		},
		{
			name:     "preseason week 2 -- still offseason mode",
			state:    map[string]interface{}{"season_type": "pre", "week": 2.0},
			expected: true,
		},
		{
			name:     "preseason week 3 (last one, ~1 week from kickoff) -- not offseason mode",
			state:    map[string]interface{}{"season_type": "pre", "week": 3.0},
			expected: false,
		},
		{
			name:     "regular season -- not offseason",
			state:    map[string]interface{}{"season_type": "regular", "week": 5.0},
			expected: false,
		},
		{
			name:     "postseason -- not offseason",
			state:    map[string]interface{}{"season_type": "post", "week": 19.0},
			expected: false,
		},
		{
			name:     "missing season_type -- assume not offseason",
			state:    map[string]interface{}{},
			expected: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isOffseasonMode(c.state); got != c.expected {
				t.Errorf("isOffseasonMode(%v) = %v, want %v", c.state, got, c.expected)
			}
		})
	}
}
