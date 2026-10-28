package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

// The spec in api/openapi.yaml is the contract with the app. TestSpec checks
// it is a valid OpenAPI document that the server serves as written, and the
// harness checks every answer the API gives in these tests against it.

const specFile = "../../api/openapi.yaml"

// loadSpec is the spec embedded in oapi.gen.go, which the server routes and
// validates requests with.
var loadSpec = sync.OnceValues(func() (*openapi3.T, error) {
	doc, err := GetSpec()
	if err != nil {
		return nil, err
	}
	return doc, doc.Validate(context.Background())
})

var specRouter = sync.OnceValues(func() (routers.Router, error) {
	doc, err := loadSpec()
	if err != nil {
		return nil, err
	}
	return gorillamux.NewRouter(doc)
})

func spec(t *testing.T) (*openapi3.T, routers.Router) {
	t.Helper()
	doc, err := loadSpec()
	if err != nil {
		t.Fatalf("embedded spec: %v", err)
	}
	router, err := specRouter()
	if err != nil {
		t.Fatalf("route the embedded spec: %v", err)
	}
	return doc, router
}

func TestSpec(t *testing.T) {
	doc, _ := spec(t)
	if doc.Info.Version != strconv.Itoa(APIVersion) {
		t.Errorf("info.version is %q, want APIVersion %d", doc.Info.Version, APIVersion)
	}
	if len(doc.Servers) != 1 || doc.Servers[0].URL != basePath {
		t.Errorf("servers = %v, want only %s", doc.Servers, basePath)
	}

	// The server is generated from the file as it is now. oapi-codegen
	// capitalizes the operation ids as it embeds the spec, and that is the
	// only difference.
	file, err := openapi3.NewLoader().LoadFromFile(specFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range file.Paths.Map() {
		for _, op := range item.Operations() {
			op.OperationID = strings.ToUpper(op.OperationID[:1]) + op.OperationID[1:]
		}
	}
	written, err := file.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	embedded, err := doc.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(written, embedded) {
		t.Errorf("oapi.gen.go was generated from another version of %s; run make generate", specFile)
	}
}

// TestSpecSecurity checks the spec decides which operations need a session:
// without one, every operation marked security: [] answers, and every other
// one is a 401 before its request is even looked at.
func TestSpecSecurity(t *testing.T) {
	h := newHarness(t)
	doc, _ := spec(t)
	param := regexp.MustCompile(`\{[^}]+\}`)
	var public int
	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			rec := h.request(method, basePath+param.ReplaceAllString(path, "1"), nil, "")
			if op.Security != nil && len(*op.Security) == 0 {
				public++
				if rec.Code == http.StatusUnauthorized {
					t.Errorf("%s (public) answered 401 without a session", op.OperationID)
				}
			} else if rec.Code != http.StatusUnauthorized {
				t.Errorf("%s answered %d without a session, want 401", op.OperationID, rec.Code)
			}
		}
	}
	if public != 2 {
		t.Errorf("%d public operations, want info and login", public)
	}

	// The spec itself is public too, for docs viewers.
	rec := h.request(http.MethodGet, basePath+"/openapi.json", nil, "")
	served, _ := GetSpecJSON()
	if rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), served) || rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("openapi.json without a session: %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
}

// checkAnswer validates an answer against the spec. Its status has to be one
// the operation lists, and its content type, headers and body have to match
// what it says for that status. A request that matches no operation has to
// be answered with an ErrorResponse: a 401 without a session, otherwise a 405
// for a known path and a 404 for any other.
func checkAnswer(t *testing.T, req *http.Request, rec *httptest.ResponseRecorder) {
	t.Helper()
	doc, router := spec(t)
	if req.URL.Path == basePath+"/openapi.json" {
		return
	}
	route, params, err := router.FindRoute(req)
	if err != nil {
		want, response := http.StatusNotFound, "NotFound"
		if errors.Is(err, routers.ErrMethodNotAllowed) {
			want, response = http.StatusMethodNotAllowed, "MethodNotAllowed"
		}
		if rec.Code == http.StatusUnauthorized {
			want, response = http.StatusUnauthorized, "Unauthorized"
		}
		content := doc.Components.Responses[response].Value.Content.Get(rec.Header().Get("Content-Type"))
		var body any
		switch {
		case rec.Code != want:
			t.Errorf("%s %s matches no operation, and answered %d, want %d", req.Method, req.URL.Path, rec.Code, want)
		case content == nil:
			t.Errorf("%s %s matches no operation, and answered %d with Content-Type %q", req.Method, req.URL.Path, rec.Code, rec.Header().Get("Content-Type"))
		case json.Unmarshal(rec.Body.Bytes(), &body) != nil:
			t.Errorf("%s %s matches no operation, and answered %d with %s", req.Method, req.URL.Path, rec.Code, rec.Body.String())
		default:
			if err := content.Schema.Value.VisitJSON(body); err != nil {
				t.Errorf("%s %s matches no operation, and answered %d with %s: %v", req.Method, req.URL.Path, rec.Code, rec.Body.String(), err)
			}
		}
		return
	}
	input := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: params,
			Route:      route,
		},
		Status:  rec.Code,
		Header:  rec.Header(),
		Options: &openapi3filter.Options{IncludeResponseStatus: true},
	}
	input.SetBodyBytes(bytes.Clone(rec.Body.Bytes()))
	if err := openapi3filter.ValidateResponse(req.Context(), input); err != nil {
		t.Errorf("%s %s (%s) answered %d against the spec: %v", req.Method, req.URL.Path, route.Operation.OperationID, rec.Code, err)
	}
}
