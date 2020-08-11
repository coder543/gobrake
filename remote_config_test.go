package gobrake

import (
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NewRemoteSettings", func() {
	var rc *remoteConfig

	BeforeEach(func() {
		rc = newRemoteConfig(&NotifierOptions{
			ProjectId:  1,
			ProjectKey: "key",
			Host:       "https://api.example.com",
			APMHost:    "https://apm.example.com",
		})
	})

	Describe("Poll", func() {
		Context("when the config is fetched", func() {
			BeforeEach(func() {
				s3Handler := func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(`{"poll_sec":10}`))
					Expect(err).To(BeNil())
				}
				s3Server := httptest.NewServer(http.HandlerFunc(s3Handler))
				rc.baseURL = s3Server.URL
			})

			It("sets a poller", func() {
				Expect(rc.poller).To(BeNil())
				rc.Poll(func(rc *remoteConfig) {})
				Expect(rc.poller).NotTo(BeNil())
			})

			It("calls the callback", func() {
				rc.Poll(func(rc *remoteConfig) {
					Expect(rc.Interval()).To(Equal(10 * time.Second))
				})
			})
		})

		Context("when the config cannot be fetched", func() {
			BeforeEach(func() {
				s3Handler := func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(`{}`))
					Expect(err).To(BeNil())
				}
				s3Server := httptest.NewServer(http.HandlerFunc(s3Handler))
				rc.baseURL = s3Server.URL
			})

			It("calls the callback with the default value", func() {
				rc.Poll(func(rc *remoteConfig) {
					Expect(rc.Interval()).To(Equal(10 * time.Minute))
				})
			})
		})
	})

	Describe("Interval", func() {
		Context("when JSON PollSec is zero", func() {
			It("returns the default interval", func() {
				interval := 600 * time.Second
				Expect(rc.Interval()).To(Equal(interval))
			})
		})

		Context("when JSON PollSec less than zero", func() {
			BeforeEach(func() {
				rc.JSON.PollSec = -123
			})

			It("returns the default interval", func() {
				interval := 600 * time.Second
				Expect(rc.Interval()).To(Equal(interval))
			})
		})

		Context("when JSON PollSec is above zero", func() {
			BeforeEach(func() {
				rc.JSON.PollSec = 123
			})

			It("returns the interval from JSON", func() {
				interval := 123 * time.Second
				Expect(rc.Interval()).To(Equal(interval))
			})
		})
	})

	Describe("ConfigRoute", func() {
		Context("when JSON ConfigRoute is empty", func() {
			It("returns the default config route", func() {
				url := "https://v1-staging-notifier-configs" +
					".s3.amazonaws.com/2020-06-18/config/" +
					"1/config.json"
				Expect(rc.ConfigRoute()).To(Equal(url))
			})
		})

		Context("when JSON ConfigRoute is non-empty", func() {
			BeforeEach(func() {
				rc.JSON.ConfigRoute = "http://example.com"
			})

			It("returns the config route from JSON", func() {
				url := "http://example.com/2020-06-18/config/" +
					"1/config.json"
				Expect(rc.ConfigRoute()).To(Equal(url))
			})
		})
	})

	Describe("EnabledErrorNotifications", func() {
		Context("when JSON settings has the 'apm' setting", func() {
			BeforeEach(func() {
				rc.JSON.RemoteSettings = append(
					rc.JSON.RemoteSettings,
					&RemoteSettings{
						Name: "errors",
					},
				)
			})

			Context("and when it is enabled", func() {
				BeforeEach(func() {
					rc.JSON.RemoteSettings[0].Enabled = true
				})

				It("returns true", func() {
					Expect(rc.EnabledErrorNotifications()).To(
						BeTrue(),
					)
				})
			})

			Context("and when it is disabled", func() {
				BeforeEach(func() {
					rc.JSON.RemoteSettings[0].Enabled = false
				})

				It("returns false", func() {
					Expect(rc.EnabledErrorNotifications()).To(
						BeFalse(),
					)
				})
			})
		})

		Context("when JSON settings has no 'apm' setting", func() {
			BeforeEach(func() {
				rc.opt.DisableErrorNotifications = false
			})

			It("returns the value from options", func() {
				Expect(rc.EnabledErrorNotifications()).To(
					BeTrue(),
				)
			})
		})
	})

	Describe("EnabledAPM", func() {
		Context("when JSON settings has the 'apm' setting", func() {
			BeforeEach(func() {
				rc.JSON.RemoteSettings = append(
					rc.JSON.RemoteSettings,
					&RemoteSettings{
						Name: "apm",
					},
				)
			})

			Context("and when it is enabled", func() {
				BeforeEach(func() {
					rc.JSON.RemoteSettings[0].Enabled = true
				})

				It("returns true", func() {
					Expect(rc.EnabledAPM()).To(BeTrue())
				})
			})

			Context("and when it is disabled", func() {
				BeforeEach(func() {
					rc.JSON.RemoteSettings[0].Enabled = false
				})

				It("returns false", func() {
					Expect(rc.EnabledAPM()).To(BeFalse())
				})
			})
		})

		Context("when JSON settings has no 'apm' setting", func() {
			BeforeEach(func() {
				rc.opt.DisableAPM = false
			})

			It("returns the value from options", func() {
				Expect(rc.EnabledAPM()).To(BeTrue())
			})
		})
	})

	Describe("ErrorHost", func() {
		Context("when JSON settings has the 'errors' setting", func() {
			BeforeEach(func() {
				rc.JSON.RemoteSettings = append(
					rc.JSON.RemoteSettings,
					&RemoteSettings{
						Name: "errors",
					},
				)
			})

			Context("and when an endpoint is specified", func() {
				BeforeEach(func() {
					setting := rc.JSON.RemoteSettings[0]
					setting.Endpoint = "http://api.newexample.com"
				})

				It("returns the endpoint", func() {
					Expect(rc.ErrorHost()).To(
						Equal("http://api.newexample.com"),
					)
				})
			})

			Context("and when an endpoint is NOT specified", func() {
				It("returns default host", func() {
					Expect(rc.ErrorHost()).To(Equal("https://api.example.com"))
				})
			})
		})

		Context("when JSON settings has no 'errors' setting", func() {
			It("returns default host", func() {
				Expect(rc.ErrorHost()).To(Equal("https://api.example.com"))
			})
		})
	})

	Describe("APMHost", func() {
		Context("when JSON settings has the 'apm' setting", func() {
			BeforeEach(func() {
				rc.JSON.RemoteSettings = append(
					rc.JSON.RemoteSettings,
					&RemoteSettings{
						Name: "apm",
					},
				)
			})

			Context("and when an endpoint is specified", func() {
				BeforeEach(func() {
					setting := rc.JSON.RemoteSettings[0]
					setting.Endpoint = "https://apm.newexample.com"
				})

				It("returns the endpoint", func() {
					Expect(rc.APMHost()).To(
						Equal("https://apm.newexample.com"),
					)
				})
			})

			Context("and when an endpoint is NOT specified", func() {
				It("returns default host", func() {
					Expect(rc.APMHost()).To(Equal("https://apm.example.com"))
				})
			})
		})

		Context("when JSON settings has no 'apm' setting", func() {
			It("returns default host", func() {
				Expect(rc.APMHost()).To(Equal("https://apm.example.com"))
			})
		})
	})
})
