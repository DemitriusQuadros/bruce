package handler

import "net/http"

type Configuration struct {
	Pattern string
	Method  string
	Action  func(http.ResponseWriter, *http.Request)
}
