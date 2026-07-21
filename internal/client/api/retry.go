package api

import (
	"net/http"
	"time"
)

const (
	maxRetries   = 3
	retryBackoff = 500 * time.Millisecond
)

// retryRoundTripper — обёртка над http.RoundTripper с автоматическими
// ретраями. Логика ретраев прозрачна для кода отправки запросов:
// он не меняется, повторные попытки происходят на уровне транспорта.
type retryRoundTripper struct {
	next http.RoundTripper
}

// RoundTrip выполняет запрос с ретраями: до maxRetries повторов
// при сетевых ошибках и ответах 429/5xx, с экспоненциальной задержкой.
func (rt *retryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; ; attempt++ {
		if attempt > 0 {
			// Тело запроса уже прочитано предыдущей попыткой — восстанавливаем.
			if req.GetBody != nil {
				body, bodyErr := req.GetBody()
				if bodyErr != nil {
					return resp, err
				}
				req.Body = body
			}
			time.Sleep(retryBackoff * time.Duration(1<<(attempt-1)))
		}

		resp, err = rt.next.RoundTrip(req)
		if attempt >= maxRetries || !shouldRetry(resp, err) {
			return resp, err
		}
		if resp != nil {
			resp.Body.Close()
		}
	}
}

// shouldRetry сообщает, имеет ли смысл повторить запрос:
// сетевая ошибка, слишком много запросов или ошибка на стороне сервера.
func shouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	return resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
}
