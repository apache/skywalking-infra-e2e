package verify

import "testing"

func TestCheckForDuplicates(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		wantErr bool
	}{
		{"no duplicates", []string{"a", "b", "c"}, false},
		{"has duplicates", []string{"a", "b", "a"}, true},
	}

	for _, tt := range tests {
		err := CheckForDuplicates(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: expected error=%v, got %v", tt.name, tt.wantErr, err != nil)
		}
	}
}
