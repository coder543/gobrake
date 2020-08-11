package gobrake_test

import (
	"net/http"
	"net/http/httptest"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/airbrake/gobrake/v4"
)

var _ = Describe("Notifier", func() {
	var notifier *gobrake.Notifier

	BeforeEach(func() {
		handler := func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, err := w.Write([]byte(`{"id":"123"}`))
			if err != nil {
				panic(err)
			}
		}
		server := httptest.NewServer(http.HandlerFunc(handler))

		s3Handler := func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{}`))
			Expect(err).To(BeNil())
		}
		s3Server := httptest.NewServer(http.HandlerFunc(s3Handler))

		notifier = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
			ProjectId:           1,
			ProjectKey:          "key",
			Host:                server.URL,
			RemoteConfigBaseURL: s3Server.URL,
		})
	})

	It("is race free", func() {
		var wg sync.WaitGroup
		for i := 0; i < 1000; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				notifier.Notify("hello", nil)
			}()
		}
		wg.Wait()

		notifier.Flush()
	})
})
