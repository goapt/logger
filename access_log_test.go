package logger

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var testreq = httptest.NewRequest("POST", "/api/test?id=1", strings.NewReader("foo"))

func init() {
	testreq.Header.Add("RequestId", "xxxx-xx-xxx-xx")
}

func TestLogFilter(t *testing.T) {
	entry := NewAccessLog(testreq)
	entry.RequestFilter = func(body []byte, r *http.Request) []byte {

		if r.RequestURI != "/upload" {
			return body
		}

		m := make(map[string]interface{})
		err := json.Unmarshal(body, &m)
		if err != nil {
			t.Fatal(err)
		}

		delete(m, "data")

		buf, err := json.Marshal(m)

		if err != nil {
			t.Fatal(err)
		}

		return buf
	}

	entry.Get()
}

func TestResponseLogFilter(t *testing.T) {
	entry := NewAccessLog(testreq)
	entry.ResponseFilter = func(body []byte, r *http.Request) []byte {
		if r.RequestURI != "/upload" {
			return body
		}

		m := make(map[string]interface{})
		err := json.Unmarshal(body, &m)
		if err != nil {
			t.Fatal(err)
		}

		delete(m, "data")

		buf, err := json.Marshal(m)

		if err != nil {
			t.Fatal(err)
		}

		return buf
	}
	entry.Get()
}
