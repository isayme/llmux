package handler

import (
	"net/http"
	"testing"
	"time"

	"llmux/internal/config"
)

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		provider   *config.ProviderConfig
		canRetry   bool
		want       bool
	}{
		{
			name:       "429 with On429=true",
			statusCode: 429,
			provider:   &config.ProviderConfig{Retry: &config.RetryConfig{On429: true}},
			canRetry:   true,
			want:       true,
		},
		{
			name:       "429 with On429=false",
			statusCode: 429,
			provider:   &config.ProviderConfig{Retry: &config.RetryConfig{On429: false}},
			canRetry:   true,
			want:       false,
		},
		{
			name:       "400 never retry",
			statusCode: 400,
			provider:   &config.ProviderConfig{Retry: &config.RetryConfig{On4xx: true}},
			canRetry:   true,
			want:       false,
		},
		{
			name:       "401 never retry",
			statusCode: 401,
			provider:   &config.ProviderConfig{Retry: &config.RetryConfig{On4xx: true}},
			canRetry:   true,
			want:       false,
		},
		{
			name:       "403 never retry",
			statusCode: 403,
			provider:   &config.ProviderConfig{Retry: &config.RetryConfig{On4xx: true}},
			canRetry:   true,
			want:       false,
		},
		{
			name:       "429 with canRetry=false",
			statusCode: 429,
			provider:   &config.ProviderConfig{Retry: &config.RetryConfig{On429: true}},
			canRetry:   false,
			want:       false,
		},
		{
			name:       "429 with nil retry config",
			statusCode: 429,
			provider:   &config.ProviderConfig{Retry: nil},
			canRetry:   true,
			want:       false,
		},
		{
			name:       "500 with On5xx=true",
			statusCode: 500,
			provider:   &config.ProviderConfig{Retry: &config.RetryConfig{On5xx: true}},
			canRetry:   true,
			want:       true,
		},
		{
			name:       "500 with On5xx=false",
			statusCode: 500,
			provider:   &config.ProviderConfig{Retry: &config.RetryConfig{On5xx: false}},
			canRetry:   true,
			want:       false,
		},
		{
			name:       "404 with On4xx=true",
			statusCode: 404,
			provider:   &config.ProviderConfig{Retry: &config.RetryConfig{On4xx: true}},
			canRetry:   true,
			want:       true,
		},
		{
			name:       "200 no retry",
			statusCode: 200,
			provider:   &config.ProviderConfig{Retry: &config.RetryConfig{On4xx: true, On5xx: true}},
			canRetry:   true,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRetry(tt.statusCode, tt.provider, tt.canRetry)
			if got != tt.want {
				t.Errorf("shouldRetry(%d, %v, %v) = %v, want %v",
					tt.statusCode, tt.provider.Retry, tt.canRetry, got, tt.want)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name   string
		header http.Header
		want   time.Duration
	}{
		{
			name:   "empty header",
			header: http.Header{},
			want:   1 * time.Second,
		},
		{
			name:   "seconds",
			header: http.Header{"Retry-After": []string{"5"}},
			want:   5 * time.Second,
		},
		{
			name:   "invalid value",
			header: http.Header{"Retry-After": []string{"invalid"}},
			want:   1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRetryAfter(tt.header)
			if got != tt.want {
				t.Errorf("parseRetryAfter() = %v, want %v", got, tt.want)
			}
		})
	}
}
