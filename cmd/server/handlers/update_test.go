package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/and161185/metrics-alerting/storage"
	"github.com/magiconair/properties/assert"
)

func TestUpdateMetricHandler(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		url        string
		wantStatus int
	}{
		{"invalid method", http.MethodGet, "/update/gauge/test/1.23", http.StatusMethodNotAllowed},
		{"valid gauge", http.MethodPost, "/update/gauge/test/1.23", http.StatusOK},
		{"valid counter", http.MethodPost, "/update/counter/testCounter/1", http.StatusOK},
		{"invalid counter value", http.MethodPost, "/update/counter/testCounter/1.2", http.StatusBadRequest},
		{"invalid type", http.MethodPost, "/update/type/testCounter/1", http.StatusBadRequest},
		{"invalid url", http.MethodPost, "/update/gauge/gauge", http.StatusNotFound},
	}

	for _, v := range tests {
		t.Run(v.name, func(t *testing.T) {
			st := storage.NewMemStorage()
			handler := UpdateMetricHandler(st)

			r := httptest.NewRequest(v.method, v.url, nil)
			w := httptest.NewRecorder()

			handler(w, r)

			response := w.Result()

			defer response.Body.Close()

			assert.Equal(t, v.wantStatus, response.StatusCode)

		})
	}
}
