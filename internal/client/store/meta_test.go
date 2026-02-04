package store

import "testing"

func TestParseMeta(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{name: "empty", input: "", want: map[string]string{}},
		{name: "single", input: "a=b", want: map[string]string{"a": "b"}},
		{name: "multiple", input: "a=b, c=d", want: map[string]string{"a": "b", "c": "d"}},
		{name: "no value", input: "a=", want: map[string]string{"a": ""}},
		{name: "missing key", input: "=b, a=c", want: map[string]string{"a": "c"}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ParseMeta(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("ParseMeta len=%d want=%d", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Fatalf("ParseMeta[%q]=%q want=%q", k, got[k], v)
				}
			}
		})
	}
}
