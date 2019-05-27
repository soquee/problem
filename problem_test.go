package problem_test

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"code.soquee.net/problem"
	"code.soquee.net/testlog"
)

var errResponderTestCases = [...]struct {
	method    string
	err       error
	code      int
	writeCode int
	body      string
}{
	0: {code: 200},
	1: {err: errors.New("test"), code: 500, body: "{}"},
	2: {err: problem.Problem{Status: 123}, code: 123, body: `{"status":123}`},
	3: {err: struct {
		problem.Problem
		Extra string `json:"ext"`
	}{
		Problem: problem.Problem{Status: 456},
		Extra:   "foo",
	}, code: 456, body: `{"status":456,"ext":"foo"}`},
	4: {err: problem.Status(http.StatusNotFound), code: http.StatusNotFound, body: `{"title":"Not Found","status":404}`},
	5: {err: problem.Problem{Title: "foo"}, code: 500, body: `{"title":"foo"}`},
	6: {err: problem.Problem{Status: -1}, writeCode: 765, code: 765, body: `{"status":-1}`},
	7: {method: "HEAD", err: errors.New("test"), code: 500, body: ""},
}

func TestErrorResponder(t *testing.T) {
	for i, tc := range errResponderTestCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if p, ok := tc.err.(problem.Problem); ok && p.Error() != p.Title {
				t.Errorf("Error should return title: want=%q, got=%q", p.Title, p.Error())
			}

			if tc.method == "" {
				tc.method = "GET"
			}
			req := httptest.NewRequest(tc.method, "/", nil)
			recorder := httptest.NewRecorder()
			if tc.writeCode != 0 {
				recorder.WriteHeader(tc.writeCode)
			}
			problem.NewResponder(testlog.New(t))(recorder, req, tc.err)

			contentType := recorder.Header().Get("Content-Type")
			if tc.method == "HEAD" && contentType != "" {
				t.Errorf("Did not expect content type for HEAD method: Content-Type: %q", contentType)
			}
			if tc.code != recorder.Code {
				t.Errorf("Unexpected status code: want=%d, got=%d", tc.code, recorder.Code)
			}
			if body := strings.TrimSpace(recorder.Body.String()); body != tc.body {
				t.Errorf("Unexpected body: want=%q, got=%q", tc.body, body)
			}
		})
	}
}

type unmarshalableErrType chan struct{}

func (unmarshalableErrType) Error() string {
	return "This is a weird error type that doesn't make sense."
}

func TestErrorResponderBadEncode(t *testing.T) {
	b := new(bytes.Buffer)
	problem.NewResponder(log.New(b, "", 0))(
		httptest.NewRecorder(),
		httptest.NewRequest("GET", "/", nil),
		make(unmarshalableErrType))

	if b.Len() == 0 {
		t.Errorf("Expected error responder to log errors encountered during marshaling")
	}
}
