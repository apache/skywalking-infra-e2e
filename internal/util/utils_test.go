package util

import "testing"

func TestExecuteCommand(t *testing.T) {
	tests := []struct {
		name    string
		cmd     string
		wantErr bool
	}{
		{
			name:    "without args",
			cmd:     "swctl",
			wantErr: false,
		},
		{
			name:    "with args",
			cmd:     "swctl service ls",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExecuteCommand(tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
