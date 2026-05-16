package router

import (
	"path"
	"strings"
)

func normalise(p string) string {
	if p == "" {
		return "/"
	}
	return strings.ToLower(path.Clean(p))
}

func matchPath(reqPath, pattern string) bool {
	reqPath = normalise(reqPath)
	pattern = normalise(pattern)

	if pattern == "*" {
		return true
	}

	// route prefix wildcard: /prefix/*
	if strings.HasSuffix(pattern, "/*") && strings.Count(pattern, "*") == 1 {
		prefix := strings.TrimSuffix(pattern, "*")
		return reqPath == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(reqPath, prefix)
	}

	// contains: *path*
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") && strings.Count(pattern, "*") == 2 {
		return strings.Contains(reqPath, pattern[1:len(pattern)-1])
	}

	// suffix: *path
	if strings.HasPrefix(pattern, "*") && strings.Count(pattern, "*") == 1 {
		return strings.HasSuffix(reqPath, pattern[1:])
	}

	// prefix: path*
	if strings.HasSuffix(pattern, "*") && strings.Count(pattern, "*") == 1 {
		return strings.HasPrefix(reqPath, pattern[:len(pattern)-1])
	}

	// exact fallback
	ok, _ := path.Match(pattern, reqPath)
	return ok
}
