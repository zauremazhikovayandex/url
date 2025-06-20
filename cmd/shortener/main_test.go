package main

import (
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/zauremazhikovayandex/url/internal/app"
	"github.com/zauremazhikovayandex/url/internal/config"
	"github.com/zauremazhikovayandex/url/internal/db/storage"
	"log"
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
		expectedBody string
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
	config.InitConfig()
	storage.InitStorage()

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
	log.Println(shortIDToOriginal)

	for id := range shortIDToOriginal {
		req := resty.New().R().
			SetHeader("Content-Type", "text/plain")
		resp, err := req.Get(bURL + "/" + id)

		assert.NoError(t, err, "error making HTTP request")
		assert.Equal(t, http.StatusOK, resp.StatusCode(), "Response code didn't match expected")
	}

}
