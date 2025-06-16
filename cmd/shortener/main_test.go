package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/zauremazhikovayandex/url/cmd/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebhook(t *testing.T) {
	type TestCase struct {
		method       string
		body         string
		expectedCode int
	}

	testCases := []TestCase{
		{
			method:       http.MethodPost,
			body:         "https://practicum.yandex.ru/",
			expectedCode: http.StatusCreated,
		},
		{
			method:       http.MethodPost,
			body:         "https://yandex.ru/",
			expectedCode: http.StatusCreated,
		},
	}

	config.AppConfig = &config.Config{
		ServerAddr: ":8080",
		BaseURL:    "http://localhost:8080",
	}

	shortIDToOriginal := make(map[string]string)

	for _, tc := range testCases {
		r := httptest.NewRequest(tc.method, "/", strings.NewReader(tc.body))
		r.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()

		PostHandler(w, r)

		resp := w.Result()
		assert.Equal(t, tc.expectedCode, resp.StatusCode, "Response code didn't match expected")

		body := w.Body.String()
		assert.True(t, strings.HasPrefix(body, "http://localhost:8080/"))

		id := strings.TrimPrefix(strings.TrimSpace(body), "http://localhost:8080/")
		shortIDToOriginal[id] = tc.body
	}

	for id, originalURL := range shortIDToOriginal {
		r := httptest.NewRequest(http.MethodGet, "/"+id, nil)
		w := httptest.NewRecorder()

		GetHandler(w, r)

		resp := w.Result()
		assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode, "Response code didn't match expected")
		assert.Equal(t, originalURL, resp.Header.Get("Location"))
	}
}
