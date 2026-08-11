package router

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const paramsKey contextKey = "route_params"

type Middleware func(http.Handler) http.Handler

type Router struct {
	routes     []route
	middleware []Middleware
}

type route struct {
	method  string
	pattern string
	parts   []string
	handler http.Handler
}

func New() *Router {
	return &Router{}
}

func (r *Router) Use(m Middleware) {
	r.middleware = append(r.middleware, m)
}

func (r *Router) Handle(method, pattern string, h http.Handler) {
	for i := len(r.middleware) - 1; i >= 0; i-- {
		h = r.middleware[i](h)
	}
	r.routes = append(r.routes, route{method: method, pattern: pattern, parts: split(pattern), handler: h})
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	for _, candidate := range r.routes {
		if candidate.method != req.Method {
			continue
		}
		params, ok := match(candidate.parts, split(req.URL.Path))
		if !ok {
			continue
		}
		ctx := context.WithValue(req.Context(), paramsKey, params)
		candidate.handler.ServeHTTP(w, req.WithContext(ctx))
		return
	}
	http.NotFound(w, req)
}

func Param(r *http.Request, name string) string {
	params, _ := r.Context().Value(paramsKey).(map[string]string)
	return params[name]
}

func split(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func match(pattern, actual []string) (map[string]string, bool) {
	if len(pattern) != len(actual) {
		return nil, false
	}
	params := map[string]string{}
	for i := range pattern {
		if strings.HasPrefix(pattern[i], "{") && strings.HasSuffix(pattern[i], "}") {
			params[strings.TrimSuffix(strings.TrimPrefix(pattern[i], "{"), "}")] = actual[i]
			continue
		}
		if pattern[i] != actual[i] {
			return nil, false
		}
	}
	return params, true
}
