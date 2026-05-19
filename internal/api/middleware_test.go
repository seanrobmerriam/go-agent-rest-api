package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAPIKey(t *testing.T) {
	t.Parallel()

	sentinel := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name       string
		configKey  string
		authHeader string
		wantStatus int
	}{
		{
			name:       "dev mode: empty key allows any request",
			configKey:  "",
			authHeader: "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "dev mode: empty key allows request with arbitrary header",
			configKey:  "",
			authHeader: "Bearer whatever",
			wantStatus: http.StatusOK,
		},
		{
			name:       "key set: correct bearer token",
			configKey:  "secret",
			authHeader: "Bearer secret",
			wantStatus: http.StatusOK,
		},
		{
			name:       "key set: missing Authorization header",
			configKey:  "secret",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "key set: wrong bearer token",
			configKey:  "secret",
			authHeader: "Bearer wrong",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "key set: raw key without Bearer prefix",
			configKey:  "secret",
			authHeader: "secret",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "key set: empty bearer value",
			configKey:  "secret",
			authHeader: "Bearer ",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "key set: different scheme",
			configKey:  "secret",
			authHeader: "Token secret",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mw := requireAPIKey(tc.configKey)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			rec := httptest.NewRecorder()
			mw(sentinel).ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}

			if tc.wantStatus == http.StatusUnauthorized {
				env := decodeEnvelope(t, rec)
				if env.Error == nil || env.Error.Code != ErrUnauthorized {
					t.Fatalf("expected UNAUTHORIZED error code, got %+v", env.Error)
				}
			}
		})
	}
}
