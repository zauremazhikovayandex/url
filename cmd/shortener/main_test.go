package main

import (
	"encoding/json"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/zauremazhikovayandex/url/internal/app"
	"github.com/zauremazhikovayandex/url/internal/config"
	"github.com/zauremazhikovayandex/url/internal/db/storage"
	"github.com/zauremazhikovayandex/url/internal/logger"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebhook(t *testing.T) {

	// Тест для String body

	testCases := []struct {
		method       string
		body         string
		expectedCode int
		expectedBody string
	}{
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
	config.InitConfig()
	storage.InitStorage()
	logger.New("info")

	srv := httptest.NewServer(app.Router())
	bURL := srv.URL
	defer srv.Close()

	shortIDToOriginal := make(map[string]string)

	for _, tc := range testCases {
		req := resty.New().R().
			SetBody(tc.body).
			SetHeader("Content-Type", "text/plain")

		resp, err := req.Post(bURL)

		assert.NoError(t, err, "error making HTTP request")
		assert.Equal(t, tc.expectedCode, resp.StatusCode(), "Response code didn't match expected")

		shortURL := strings.TrimSpace(string(resp.Body()))
		assert.True(t, strings.HasPrefix(shortURL, "http://localhost:8080/"))

		id := strings.TrimPrefix(shortURL, "http://localhost:8080/")
		shortIDToOriginal[id] = tc.body
	}

	// Тест для JSON body

	jsonCases := []struct {
		jsonBody     map[string]string
		expectedCode int
	}{
		{
			jsonBody:     map[string]string{"url": "https://go.dev"},
			expectedCode: http.StatusCreated,
		},
		{
			jsonBody:     map[string]string{"url": "https://example.com"},
			expectedCode: http.StatusCreated,
		},
	}

	for _, tc := range jsonCases {
		req := resty.New().R().
			SetHeader("Content-Type", "application/json").
			SetBody(tc.jsonBody)

		resp, err := req.Post(bURL + "/api/shorten")
		assert.NoError(t, err, "error making JSON request")
		assert.Equal(t, tc.expectedCode, resp.StatusCode(), "JSON response code didn't match expected")

		var result struct {
			Result string `json:"result"`
		}
		err = json.Unmarshal(resp.Body(), &result)
		assert.NoError(t, err, "Failed to unmarshal JSON response")

		assert.True(t, strings.HasPrefix(result.Result, "http://localhost:8080/"))

		id := strings.TrimPrefix(result.Result, "http://localhost:8080/")
		shortIDToOriginal[id] = tc.jsonBody["url"]
	}

	// Тест Get запрос

	for id := range shortIDToOriginal {
		req := resty.New().R().
			SetHeader("Content-Type", "text/plain")
		resp, err := req.Get(bURL + "/" + id)

		assert.NoError(t, err, "error making HTTP request")
		assert.Equal(t, http.StatusOK, resp.StatusCode(), "Response code didn't match expected")
	}

}
