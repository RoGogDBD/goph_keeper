package auth

import "testing"

func TestHashComparePassword(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		password string
		check    string
		wantErr  bool
	}{
		{name: "match", password: "secret", check: "secret", wantErr: false},
		{name: "mismatch", password: "secret", check: "bad", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			hash, err := HashPassword(tc.password)
			if err != nil {
				t.Fatalf("HashPassword: %v", err)
			}
			err = ComparePassword(hash, tc.check)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ComparePassword err=%v wantErr=%v", err, tc.wantErr)
			}
		})
	}
}
