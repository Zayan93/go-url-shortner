package app

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

var (
	store = make(map[string]string)
)

func generateID() string {
	b := make([]byte, 6) // 6 байт = ~8 символов base64
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

func GetPage(res http.ResponseWriter, req *http.Request) {

	if req.Method != http.MethodGet {
		http.Error(res, "bad request", http.StatusBadRequest)
		return
	}
	id := strings.TrimPrefix(req.URL.Path, "/")
	if id == "" {
		http.Error(res, "bad request", http.StatusBadRequest)
		return
	}

	originalURL, found := store[id]
	if !found {
		http.Error(res, "not found", http.StatusBadRequest)
		return
	}

	http.Redirect(res, req, originalURL, http.StatusTemporaryRedirect)

}

func PostPage(res http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		data, err := io.ReadAll(req.Body)
		if err != nil || len(data) == 0 {
			http.Error(res, "bad request", http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		originalURL := strings.TrimSpace(string(data))
		id := generateID()

		store[id] = originalURL

		if req.Header.Get("Content-Type") != "text/plain" {
			http.Error(res, "invalid content type", http.StatusBadRequest)
			return
		}
		shortURL := fmt.Sprintf("http://%s/%s", req.Host, id)
		res.Header().Set("Content-Type", "text/plain")
		res.Header().Set("Content-Length", strconv.Itoa(len(shortURL)))
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(shortURL))
	} else {
		res.WriteHeader(http.StatusBadRequest)
	}
}
