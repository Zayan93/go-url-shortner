package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-url-shortner/internal/store"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestPostPage(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name       string
		requestURL string
		want       want
	}{
		{
			name:       "positive test #1",
			requestURL: "https://practicum.yandex.ru/",
			want: want{
				code:        http.StatusCreated,
				response:    `{"status":"Created"}`,
				contentType: "text/plain",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageName := "storage1.txt"
			storage, _ := store.NewFileStorage(storageName)
			handler := NewHandler(storage, "http://localhost:8080")

			body := strings.NewReader(tt.requestURL)
			request := httptest.NewRequest(http.MethodPost, "/", body)
			request.Host = "localhost:8080"
			request.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()
			handler.PostPage(w, request)
			res := w.Result()
			// проверяем код ответа
			assert.Equal(t, tt.want.code, res.StatusCode)
			// получаем и проверяем тело запроса
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)

			require.NoError(t, err)
			assert.Contains(t, string(resBody), "http://localhost:8080/")
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			_ = storage.Close()
			if err := os.Remove(storageName); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestHandler_PostShorten(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	type reqBody struct {
		URL string `json:"url"`
	}
	tests := []struct {
		name       string
		requestURL string
		body       reqBody
		want       want
	}{
		{
			name: "positive test shorten #1",
			body: reqBody{URL: "https://practicum.yandex.ru/"},
			want: want{
				code:        http.StatusCreated,
				response:    `{"status":"Created"}`,
				contentType: "application/json",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageName := "storage2.txt"
			storage, _ := store.NewFileStorage(storageName)
			handler := NewHandler(storage, "http://localhost:8080")

			buf := new(bytes.Buffer)
			err := json.NewEncoder(buf).Encode(tt.body)
			require.NoError(t, err)

			request := httptest.NewRequest(http.MethodPost, "/api/shorten", buf)
			request.Host = "localhost:8080"
			request.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.PostShorten(w, request)
			res := w.Result()
			defer res.Body.Close()
			// проверяем код ответа
			assert.Equal(t, tt.want.code, res.StatusCode)
			// получаем и проверяем тело запроса

			resBody, err := io.ReadAll(res.Body)

			require.NoError(t, err)

			var responseBody struct {
				Result string `json:"result"`
			}
			err = json.Unmarshal(resBody, &responseBody)
			require.NoError(t, err)

			assert.True(t, strings.HasPrefix(responseBody.Result, "http://localhost:8080/"))
			_ = storage.Close()
			if err := os.Remove(storageName); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestGetPage(t *testing.T) {
	type want struct {
		code     int
		response string
		Location string
	}
	tests := []struct {
		name        string
		requestURL  string
		contentType string
		Host        string
		want        want
	}{
		{
			name:        "positive test #1",
			Host:        "localhost:8080",
			requestURL:  "https://practicum.yandex.ru/",
			contentType: "text/plain ",
			want: want{
				code:     http.StatusTemporaryRedirect,
				response: `{"status":"Created"}`,
				Location: "text/plain",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Шаг 1 для начала POST запросом созданим тест данные
			storageName := "storage3.txt"
			storage, err := store.NewFileStorage(storageName)
			require.NoError(t, err)

			handler := NewHandler(storage, "http://localhost:8080")

			body := strings.NewReader(tt.requestURL)
			postRequest := httptest.NewRequest(http.MethodPost, "/", body)
			postRequest.Host = "localhost:8080"
			postRequest.Header.Set("Content-Type", "text/plain")
			postWriter := httptest.NewRecorder()
			handler.PostPage(postWriter, postRequest)

			// Получаем id из ответа
			postRes := postWriter.Result()
			require.Equal(t, http.StatusCreated, postRes.StatusCode)
			shortURLBytes, err := io.ReadAll(postRes.Body)
			require.NoError(t, err)
			shortURL := string(shortURLBytes)
			_ = postRes.Body.Close()

			parts := strings.Split(shortURL, "/")
			id := parts[len(parts)-1]
			fmt.Println("Id from response:", id)

			// Шаг 2 запрашиваем GET запросом данные
			getRequest := httptest.NewRequest(http.MethodGet, "/"+id, nil)
			getRequest.Host = tt.Host
			getRequest.Header.Set("Content-Type", tt.contentType)
			// создаём новый Recorder
			w := httptest.NewRecorder()
			handler.GetPage(w, getRequest)

			res := w.Result()
			// проверяем код ответа
			assert.Equal(t, tt.want.code, res.StatusCode)
			// получаем и проверяем тело запроса
			defer res.Body.Close()

			assert.Equal(t, tt.requestURL, res.Header.Get("Location"))
			// удалим файл settings.json
			_ = storage.Close()
			if err := os.Remove(storageName); err != nil {
				t.Error(err)
			}

		})
	}
}
