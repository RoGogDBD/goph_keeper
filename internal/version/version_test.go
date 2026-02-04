package version

import "testing"

func TestInfo(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
	}{
		{name: "info"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			BuildVersion = "v1"
			BuildDate = "2025-01-01"
			BuildCommit = "abc"

			got := Info()
			if got == "" {
				t.Fatalf("Info is empty")
			}
		})
	}
}
