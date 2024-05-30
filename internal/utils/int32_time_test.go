package utils

import (
	"testing"
	"time"
)

func TestTimeToInt32(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected int32
		wantErr  bool
	}{
		{
			name:     "Epoch Time",
			input:    time.Unix(0, 0),
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "Some Random Now Time",
			input:    time.Unix(1716912942, 0),
			expected: 1716912942,
			wantErr:  false,
		},
		{
			name:     "Positive Time within range",
			input:    time.Unix(2147483647, 0), // Maximum int32 value
			expected: 2147483647,
			wantErr:  false,
		},
		{
			name:     "Negative Time within range",
			input:    time.Unix(-2147483648, 0), // Minimum int32 value
			expected: -2147483648,
			wantErr:  false,
		},
		{
			name:    "Time out of positive range",
			input:   time.Unix(2147483648, 0),
			wantErr: true,
		},
		{
			name:    "Time out of negative range",
			input:   time.Unix(-2147483649, 0),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := TimeToInt32(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("TimeToInt32() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("TimeToInt32() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
