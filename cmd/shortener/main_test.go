package main

import (
	"github.com/stretchr/testify/assert"
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

	for _, tc := range testCases {
		r := httptest.NewRequest(tc.method, "/", strings.NewReader(tc.body))
		r.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()

		PostHandler(w, r)
		assert.Equal(t, tc.expectedCode, w.Code, "Код ответа не совпадает с ожидаемым")
	}
}
