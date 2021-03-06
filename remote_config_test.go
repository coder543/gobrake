package gobrake

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("newRemoteConfig", func() {
	var rc *remoteConfig
	var opt *NotifierOptions
	var origLogger *log.Logger
	var logBuf *bytes.Buffer

	Describe("Poll", func() {
		BeforeEach(func() {
			opt = &NotifierOptions{
				ProjectId:  1,
				ProjectKey: "key",
			}

			origLogger = GetLogger()
			logBuf = new(bytes.Buffer)
			SetLogger(log.New(logBuf, "", 0))
		})

		JustBeforeEach(func() {
			rc = newRemoteConfig(opt)
		})

		AfterEach(func() {
			SetLogger(origLogger)
			rc.StopPolling()
		})

		Context("when the server returns 404", func() {
			BeforeEach(func() {
				handler := func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusNotFound)
					_, err := w.Write([]byte("not found"))
					Expect(err).To(BeNil())
				}
				server := httptest.NewServer(http.HandlerFunc(handler))

				opt.RemoteConfigHost = server.URL
			})

			It("logs the error", func() {
				rc.Poll()
				Expect(logBuf.String()).To(
					ContainSubstring("fetchConfig failed: not found"),
				)
			})
		})

		Context("when the server returns 403", func() {
			BeforeEach(func() {
				handler := func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusForbidden)
					_, err := w.Write([]byte("forbidden"))
					Expect(err).To(BeNil())
				}
				server := httptest.NewServer(http.HandlerFunc(handler))

				opt.RemoteConfigHost = server.URL
			})

			It("logs the error", func() {
				rc.Poll()
				Expect(logBuf.String()).To(
					ContainSubstring("fetchConfig failed: forbidden"),
				)
			})
		})

		Context("when the server returns 200", func() {
			Context("and when it returns correct config JSON", func() {
				BeforeEach(func() {
					handler := func(w http.ResponseWriter, req *http.Request) {
						w.WriteHeader(http.StatusOK)
						_, err := w.Write([]byte("{}"))
						Expect(err).To(BeNil())
					}
					server := httptest.NewServer(http.HandlerFunc(handler))

					opt.RemoteConfigHost = server.URL
				})

				It("doesn't log any errors", func() {
					rc.Poll()
					Expect(logBuf.String()).To(BeEmpty())
				})
			})

			Context("and when it returns incorrect JSON config", func() {
				BeforeEach(func() {
					handler := func(w http.ResponseWriter, req *http.Request) {
						w.WriteHeader(http.StatusOK)
						_, err := w.Write([]byte("{"))
						Expect(err).To(BeNil())
					}
					server := httptest.NewServer(http.HandlerFunc(handler))

					opt.RemoteConfigHost = server.URL
				})

				It("logs the error", func() {
					rc.Poll()
					Expect(logBuf.String()).To(
						ContainSubstring(
							"parseConfig failed: unexpected end of JSON input",
						),
					)
				})
			})

			Context("and when it returns JSON with missing config fields", func() {
				BeforeEach(func() {
					handler := func(w http.ResponseWriter, req *http.Request) {
						w.WriteHeader(http.StatusOK)
						_, err := w.Write([]byte(`{"hello":"hi"}`))
						Expect(err).To(BeNil())
					}
					server := httptest.NewServer(http.HandlerFunc(handler))

					opt.RemoteConfigHost = server.URL
				})

				It("doesn't log any errors", func() {
					rc.Poll()
					Expect(logBuf.String()).To(BeEmpty())
				})
			})

			Context("and when it returns JSON with current config fields", func() {
				var body = `{"project_id":1,"updated_at":2,` +
					`"poll_sec":3,"config_route":"abc/config.json",` +
					`"settings":[{"name":"errors","enabled":false,` +
					`"endpoint":null}]}`

				BeforeEach(func() {
					handler := func(w http.ResponseWriter, req *http.Request) {
						w.WriteHeader(http.StatusOK)
						_, err := w.Write([]byte(body))
						Expect(err).To(BeNil())
					}
					server := httptest.NewServer(http.HandlerFunc(handler))

					opt.RemoteConfigHost = server.URL
				})

				It("doesn't log any errors", func() {
					rc.Poll()
					Expect(logBuf.String()).To(BeEmpty())
				})
			})
		})

		Context("when the server returns unhandled code", func() {
			BeforeEach(func() {
				handler := func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusGone)
					_, err := w.Write([]byte("{}"))
					Expect(err).To(BeNil())
				}
				server := httptest.NewServer(http.HandlerFunc(handler))

				opt.RemoteConfigHost = server.URL
			})

			It("logs the unhandled error", func() {
				rc.Poll()
				Expect(logBuf.String()).To(
					ContainSubstring("unhandled status (410): {}"),
				)
			})
		})

		Context("when the remote config alters poll_sec", func() {
			var body = `{"poll_sec":1}`

			BeforeEach(func() {
				handler := func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(body))
					Expect(err).To(BeNil())
				}
				server := httptest.NewServer(http.HandlerFunc(handler))

				opt.RemoteConfigHost = server.URL
			})

			It("changes interval", func() {
				Expect(rc.Interval()).NotTo(Equal(1 * time.Second))
				rc.Poll()
				rc.StopPolling()
				Expect(rc.Interval()).To(Equal(1 * time.Second))
			})
		})

		Context("when the remote config alters config_route", func() {
			var body = `{"config_route":"route/cfg.json"}`

			BeforeEach(func() {
				handler := func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(body))
					Expect(err).To(BeNil())
				}
				server := httptest.NewServer(http.HandlerFunc(handler))

				opt.RemoteConfigHost = server.URL
			})

			It("changes config route", func() {
				Expect(rc.ConfigRoute("http://example.com")).NotTo(Equal(
					"http://example.com/route/cfg.json",
				))
				rc.Poll()
				rc.StopPolling()
				Expect(rc.ConfigRoute("http://example.com")).To(Equal(
					"http://example.com/route/cfg.json",
				))
			})
		})
	})

	Describe("Interval", func() {
		BeforeEach(func() {
			rc = newRemoteConfig(&NotifierOptions{
				ProjectId:  1,
				ProjectKey: "key",
			})
		})

		Context("when JSON PollSec is zero", func() {
			JustBeforeEach(func() {
				rc.JSON.PollSec = 0
			})

			It("returns the default interval", func() {
				Expect(rc.Interval()).To(Equal(600 * time.Second))
			})
		})

		Context("when JSON PollSec less than zero", func() {
			JustBeforeEach(func() {
				rc.JSON.PollSec = -123
			})

			It("returns the default interval", func() {
				Expect(rc.Interval()).To(Equal(600 * time.Second))
			})
		})

		Context("when JSON PollSec is above zero", func() {
			BeforeEach(func() {
				rc.JSON.PollSec = 123
			})

			It("returns the interval from JSON", func() {
				Expect(rc.Interval()).To(Equal(123 * time.Second))
			})
		})
	})

	Describe("ConfigRoute", func() {
		BeforeEach(func() {
			rc = newRemoteConfig(&NotifierOptions{
				ProjectId:  1,
				ProjectKey: "key",
			})
		})

		Context("when JSON ConfigRoute is empty", func() {
			BeforeEach(func() {
				rc.JSON.ConfigRoute = ""
			})

			It("returns the default config route", func() {
				Expect(rc.ConfigRoute("http://example.com")).To(Equal(
					"http://example.com/2020-06-18/config/1/config.json",
				))
			})
		})

		Context("when JSON ConfigRoute is non-empty", func() {
			BeforeEach(func() {
				rc.JSON.ConfigRoute = "1999/123/config.json"
			})

			It("returns the config route from JSON", func() {
				Expect(rc.ConfigRoute("http://example.com")).To(Equal(
					"http://example.com/1999/123/config.json",
				))
			})
		})

		Context("when given hostname ends with a dash", func() {
			It("trims the dash and returns the correct route", func() {
				host := "http://example.com/"
				Expect(rc.ConfigRoute(host)).To(Equal(
					"http://example.com/2020-06-18/config/1/config.json",
				))
			})
		})
	})
})
