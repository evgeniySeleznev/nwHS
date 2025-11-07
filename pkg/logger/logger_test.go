package logger

import "testing"

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "production_defaults",
			cfg: Config{
				Level:       "info",
				Environment: "prod",
				Encoding:    "json",
			},
		},
		{
			name: "invalid_level",
			cfg: Config{
				Level:       "verbose",
				Environment: "prod",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.cfg)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr {
				t.Cleanup(func() {
					_ = logger.Sync()
				})
			}
		})
	}
}

func TestMust(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic")
		}
	}()

	Must(Config{Level: "verbose"})
}
