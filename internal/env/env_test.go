package env

import (
	"os"
	"testing"
)

func TestGetString(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		setValue     string
		setEnv       bool
		defaultValue string
		want         string
	}{
		{
			name:         "returns value when env var is set",
			key:          "TEST_STRING",
			setValue:     "hello",
			setEnv:       true,
			defaultValue: "default",
			want:         "hello",
		},
		{
			name:         "returns default when env var is not set",
			key:          "TEST_STRING_UNSET",
			setEnv:       false,
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "returns default when env var is empty",
			key:          "TEST_STRING_EMPTY",
			setValue:     "",
			setEnv:       true,
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "handles spaces in value",
			key:          "TEST_STRING_SPACES",
			setValue:     "  hello world  ",
			setEnv:       true,
			defaultValue: "default",
			want:         "  hello world  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.setEnv {
				os.Setenv(tt.key, tt.setValue)
				defer os.Unsetenv(tt.key)
			}

			// Test
			got := GetString(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		setValue     string
		setEnv       bool
		defaultValue bool
		want         bool
	}{
		{
			name:         "returns true for '1'",
			key:          "TEST_BOOL_1",
			setValue:     "1",
			setEnv:       true,
			defaultValue: false,
			want:         true,
		},
		{
			name:         "returns true for 'true'",
			key:          "TEST_BOOL_TRUE",
			setValue:     "true",
			setEnv:       true,
			defaultValue: false,
			want:         true,
		},
		{
			name:         "returns true for 'TRUE'",
			key:          "TEST_BOOL_TRUE_UPPER",
			setValue:     "TRUE",
			setEnv:       true,
			defaultValue: false,
			want:         true,
		},
		{
			name:         "returns true for 'True'",
			key:          "TEST_BOOL_TRUE_MIXED",
			setValue:     "True",
			setEnv:       true,
			defaultValue: false,
			want:         true,
		},
		{
			name:         "returns false for '0'",
			key:          "TEST_BOOL_0",
			setValue:     "0",
			setEnv:       true,
			defaultValue: true,
			want:         false,
		},
		{
			name:         "returns false for 'false'",
			key:          "TEST_BOOL_FALSE",
			setValue:     "false",
			setEnv:       true,
			defaultValue: true,
			want:         false,
		},
		{
			name:         "returns false for 'FALSE'",
			key:          "TEST_BOOL_FALSE_UPPER",
			setValue:     "FALSE",
			setEnv:       true,
			defaultValue: true,
			want:         false,
		},
		{
			name:         "returns default when not set",
			key:          "TEST_BOOL_UNSET",
			setEnv:       false,
			defaultValue: true,
			want:         true,
		},
		{
			name:         "returns default when empty",
			key:          "TEST_BOOL_EMPTY",
			setValue:     "",
			setEnv:       true,
			defaultValue: true,
			want:         true,
		},
		{
			name:         "returns default for invalid value",
			key:          "TEST_BOOL_INVALID",
			setValue:     "invalid",
			setEnv:       true,
			defaultValue: true,
			want:         true,
		},
		{
			name:         "returns default for numeric non-boolean",
			key:          "TEST_BOOL_INVALID_NUM",
			setValue:     "2",
			setEnv:       true,
			defaultValue: false,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.setEnv {
				os.Setenv(tt.key, tt.setValue)
				defer os.Unsetenv(tt.key)
			}

			// Test
			got := GetBool(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetFloat(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		setValue     string
		setEnv       bool
		defaultValue float64
		want         float64
	}{
		{
			name:         "returns float for valid value",
			key:          "TEST_FLOAT_VALID",
			setValue:     "3.14",
			setEnv:       true,
			defaultValue: 0.0,
			want:         3.14,
		},
		{
			name:         "returns integer as float",
			key:          "TEST_FLOAT_INT",
			setValue:     "42",
			setEnv:       true,
			defaultValue: 0.0,
			want:         42.0,
		},
		{
			name:         "returns negative float",
			key:          "TEST_FLOAT_NEGATIVE",
			setValue:     "-2.5",
			setEnv:       true,
			defaultValue: 0.0,
			want:         -2.5,
		},
		{
			name:         "returns scientific notation",
			key:          "TEST_FLOAT_SCIENTIFIC",
			setValue:     "1.5e2",
			setEnv:       true,
			defaultValue: 0.0,
			want:         150.0,
		},
		{
			name:         "returns default when not set",
			key:          "TEST_FLOAT_UNSET",
			setEnv:       false,
			defaultValue: 9.99,
			want:         9.99,
		},
		{
			name:         "returns default when empty",
			key:          "TEST_FLOAT_EMPTY",
			setValue:     "",
			setEnv:       true,
			defaultValue: 1.23,
			want:         1.23,
		},
		{
			name:         "returns default for invalid value",
			key:          "TEST_FLOAT_INVALID",
			setValue:     "not-a-number",
			setEnv:       true,
			defaultValue: 5.0,
			want:         5.0,
		},
		{
			name:         "handles zero",
			key:          "TEST_FLOAT_ZERO",
			setValue:     "0.0",
			setEnv:       true,
			defaultValue: 1.0,
			want:         0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.setEnv {
				os.Setenv(tt.key, tt.setValue)
				defer os.Unsetenv(tt.key)
			}

			// Test
			got := GetFloat(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		setValue     string
		setEnv       bool
		defaultValue int
		want         int
	}{
		{
			name:         "returns int for valid value",
			key:          "TEST_INT_VALID",
			setValue:     "42",
			setEnv:       true,
			defaultValue: 0,
			want:         42,
		},
		{
			name:         "returns negative int",
			key:          "TEST_INT_NEGATIVE",
			setValue:     "-10",
			setEnv:       true,
			defaultValue: 0,
			want:         -10,
		},
		{
			name:         "returns zero",
			key:          "TEST_INT_ZERO",
			setValue:     "0",
			setEnv:       true,
			defaultValue: 100,
			want:         0,
		},
		{
			name:         "returns default when not set",
			key:          "TEST_INT_UNSET",
			setEnv:       false,
			defaultValue: 123,
			want:         123,
		},
		{
			name:         "returns default when empty",
			key:          "TEST_INT_EMPTY",
			setValue:     "",
			setEnv:       true,
			defaultValue: 456,
			want:         456,
		},
		{
			name:         "returns default for float value",
			key:          "TEST_INT_FLOAT",
			setValue:     "3.14",
			setEnv:       true,
			defaultValue: 10,
			want:         10,
		},
		{
			name:         "returns default for invalid value",
			key:          "TEST_INT_INVALID",
			setValue:     "not-a-number",
			setEnv:       true,
			defaultValue: 99,
			want:         99,
		},
		{
			name:         "handles large numbers",
			key:          "TEST_INT_LARGE",
			setValue:     "2147483647",
			setEnv:       true,
			defaultValue: 0,
			want:         2147483647,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.setEnv {
				os.Setenv(tt.key, tt.setValue)
				defer os.Unsetenv(tt.key)
			}

			// Test
			got := GetInt(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("GetInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetHostname(t *testing.T) {
	tests := []struct {
		name         string
		defaultValue string
	}{
		{
			name:         "returns hostname or default",
			defaultValue: "default-host",
		},
		{
			name:         "uses empty default",
			defaultValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetHostname(tt.defaultValue)

			// GetHostname should either return actual hostname or default
			// We can't predict the actual hostname, so verify it's not empty
			// or equals default
			if got == "" && tt.defaultValue != "" {
				t.Errorf("GetHostname() returned empty when default was %q", tt.defaultValue)
			}

			// If we got the default back, that's valid (might be an error case)
			// If we got something else, it should be non-empty (actual hostname)
			if got != tt.defaultValue && got == "" {
				t.Errorf("GetHostname() returned empty string but should return hostname or default")
			}
		})
	}
}

// Benchmark tests
func BenchmarkGetString(b *testing.B) {
	os.Setenv("BENCH_STRING", "value")
	defer os.Unsetenv("BENCH_STRING")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetString("BENCH_STRING", "default")
	}
}

func BenchmarkGetBool(b *testing.B) {
	os.Setenv("BENCH_BOOL", "true")
	defer os.Unsetenv("BENCH_BOOL")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetBool("BENCH_BOOL", false)
	}
}

func BenchmarkGetFloat(b *testing.B) {
	os.Setenv("BENCH_FLOAT", "3.14")
	defer os.Unsetenv("BENCH_FLOAT")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetFloat("BENCH_FLOAT", 0.0)
	}
}

func BenchmarkGetInt(b *testing.B) {
	os.Setenv("BENCH_INT", "42")
	defer os.Unsetenv("BENCH_INT")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetInt("BENCH_INT", 0)
	}
}
