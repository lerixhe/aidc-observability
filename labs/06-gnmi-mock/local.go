package main

import (
	"net/http"
	"net/http/httptest"
)

// newLocalServer 起一个真实 HTTP 服务，端口由系统分配。
// 用于 -get / -subscribe 这类一次性交互，避免和 -serve 抢端口。
func newLocalServer(h http.Handler) *httptest.Server {
	srv := httptest.NewUnstartedServer(h)
	srv.Start()
	return srv
}
