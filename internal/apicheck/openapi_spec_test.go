//go:build apicheck

package apicheck

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// A minimal reader for the parts of Stripe's OpenAPI spec this check needs:
// which paths exist, and which query parameters each operation declares.

type openAPISpec struct {
	version string
	paths   map[string]map[string]operation
}

type operation struct {
	params map[string]bool
}

// declares reports whether the endpoint accepts a query parameter. Stripe
// declares nested and indexed parameters by their base name (created, expand),
// so created[gte] and expand[] match created and expand.
func (o operation) declares(param string) bool {
	if o.params[param] {
		return true
	}
	base := param
	if idx := strings.IndexByte(base, '['); idx > 0 {
		base = base[:idx]
	}
	return o.params[base]
}

func (s openAPISpec) lookup(method, path string) (operation, string, bool) {
	methods, ok := s.paths[path]
	if ok {
		op, ok := methods[strings.ToLower(method)]
		return op, path, ok
	}
	// Try templated paths: /v1/charges/{charge} against /v1/charges/ch_123.
	requestSegments := strings.Split(strings.Trim(path, "/"), "/")
	for template, methods := range s.paths {
		templateSegments := strings.Split(strings.Trim(template, "/"), "/")
		if len(templateSegments) != len(requestSegments) {
			continue
		}
		match := true
		for idx, segment := range templateSegments {
			if strings.HasPrefix(segment, "{") {
				continue
			}
			if segment != requestSegments[idx] {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		if op, ok := methods[strings.ToLower(method)]; ok {
			return op, template, true
		}
	}
	return operation{}, "", false
}

func loadSpec(t *testing.T) openAPISpec {
	t.Helper()
	path := os.Getenv("STRIPE_OPENAPI_SPEC")
	if path == "" {
		t.Skip("set STRIPE_OPENAPI_SPEC to Stripe's spec3.json (see `make apicheck`)")
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open spec: %v", err)
	}
	defer file.Close()

	var raw struct {
		Info  struct{ Version string } `json:"info"`
		Paths map[string]map[string]struct {
			Parameters []struct {
				Name string `json:"name"`
				In   string `json:"in"`
			} `json:"parameters"`
		} `json:"paths"`
	}
	if err := json.NewDecoder(file).Decode(&raw); err != nil {
		t.Fatalf("decode spec: %v", err)
	}

	spec := openAPISpec{version: raw.Info.Version, paths: map[string]map[string]operation{}}
	for path, methods := range raw.Paths {
		spec.paths[path] = map[string]operation{}
		for method, op := range methods {
			params := map[string]bool{}
			for _, param := range op.Parameters {
				if param.In == "query" {
					params[param.Name] = true
				}
			}
			spec.paths[path][strings.ToLower(method)] = operation{params: params}
		}
	}
	return spec
}
