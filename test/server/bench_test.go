package servertest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	types "github.com/Shyp/go-types"
	"github.com/Shyp/rickover/server"
	"github.com/Shyp/rickover/test"
	"github.com/Shyp/rickover/test/factory"
)

func BenchmarkEnqueue(b *testing.B) {
	defer test.TearDown(b)
	expiry := time.Now().UTC().Add(5 * time.Minute)
	ejr := &server.EnqueueJobRequest{
		Data:      factory.EmptyData,
		ExpiresAt: types.NullTime{Valid: true, Time: expiry},
	}
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(ejr)
	bits := buf.Bytes()
	_ = factory.CreateJob(b, factory.SampleJob)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("PUT", "/v1/jobs/echo/random_id", bytes.NewReader(bits))
		req.SetBasicAuth("test", testPassword)
		w := httptest.NewRecorder()
		server.DefaultServer.ServeHTTP(w, req)
		b.SetBytes(int64(w.Body.Len()))
		if w.Code != 202 {
			b.Fatalf("incorrect Code: %d (response %s)", w.Code, w.Body.Bytes())
		}
	}
}
