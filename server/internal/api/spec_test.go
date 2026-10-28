package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

// The spec in api/openapi.yaml is the contract with the app. TestSpec checks
// it is a valid OpenAPI document, and the harness checks every answer the API
// gives in these tests against it.

const specFile = "../../api/openapi.yaml"

var loadSpec = sync.OnceValues(func() (*openapi3.T, error) {
	doc, err := openapi3.NewLoader().LoadFromFile(specFile)
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
		t.Fatalf("%s: %v", specFile, err)
	}
	router, err := specRouter()
	if err != nil {
		t.Fatalf("route %s: %v", specFile, err)
	}
	return doc, router
}

func TestSpec(t *testing.T) {
	doc, _ := spec(t)
	if doc.Info.Version != strconv.Itoa(APIVersion) {
		t.Errorf("info.version is %q, want APIVersion %d", doc.Info.Version, APIVersion)
	}
}

// checkAnswer validates an answer against the spec. Its status has to be one
// the operation lists, and its content type, headers and body have to match
// what it says for that status. A request that matches no operation has to be
// answered with an ErrorResponse.
func checkAnswer(t *testing.T, req *http.Request, rec *httptest.ResponseRecorder) {
	t.Helper()
	doc, router := spec(t)
	route, params, err := router.FindRoute(req)
	if err != nil {
		var body any
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("%s %s matches no operation, and answered %d with Content-Type %q", req.Method, req.URL.Path, rec.Code, ct)
		} else if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Errorf("%s %s matches no operation, and answered %d with %s", req.Method, req.URL.Path, rec.Code, rec.Body.String())
		} else if err := doc.Components.Schemas["ErrorResponse"].Value.VisitJSON(body); err != nil {
			t.Errorf("%s %s matches no operation, and answered %d with %s: %v", req.Method, req.URL.Path, rec.Code, rec.Body.String(), err)
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
