package uaa_go_client_test

import (
	"errors"
	"github.com/pivotal-golang/lager/lagertest"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cf-routing/uaa-go-client"
	"github.com/cf-routing/uaa-go-client/config"

	"github.com/pivotal-golang/clock/fakeclock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Fetch Key", func() {

	const (
		TOKEN_KEY_ENDPOINT = "/token_key"
	)
	var (
		client *uaa_go_client.UaaClient
		err    error
		key    string
	)

	Context("FetchKey", func() {
		BeforeEach(func() {
			cfg = &config.Config{
				MaxNumberOfRetries:    DefaultMaxNumberOfRetries,
				RetryInterval:         DefaultRetryInterval,
				ExpirationBufferInSec: DefaultExpirationBufferTime,
			}
			server = ghttp.NewServer()

			url, err := url.Parse(server.URL())
			Expect(err).ToNot(HaveOccurred())

			addr := strings.Split(url.Host, ":")

			cfg.UaaEndpoint = "http://" + addr[0] + ":" + addr[1]
			Expect(err).ToNot(HaveOccurred())

			cfg.ClientName = "client-name"
			cfg.ClientSecret = "client-secret"
			clock = fakeclock.NewFakeClock(time.Now())
			logger = lagertest.NewTestLogger("test")

			client, err = uaa_go_client.NewClient(logger, cfg, clock)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})

		AfterEach(func() {
			server.Close()
		})

		JustBeforeEach(func() {
			key, err = client.FetchKey()
		})

		Context("when UAA is available and responsive", func() {

			Context("and http request succeeds", func() {
				BeforeEach(func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", TOKEN_KEY_ENDPOINT),
							ghttp.RespondWith(http.StatusOK, `{}`),
						),
					)
				})
				It("does not return an error", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(key).NotTo(BeNil())
				})
			})

			Context("and returns a valid uaa key", func() {
				BeforeEach(func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", TOKEN_KEY_ENDPOINT),
							ghttp.RespondWith(http.StatusOK, `{"alg":"alg", "value": "AABBCC" }`),
						),
					)
				})

				It("returns the key value", func() {
					Expect(err).NotTo(HaveOccurred())
					Expect(key).NotTo(BeNil())
					Expect(key).Should(Equal("AABBCC"))
				})

				It("logs success message", func() {
					Expect(logger).Should(gbytes.Say("fetch-key-successful"))
				})
			})

			Context("and returns a invalid json uaa key", func() {
				BeforeEach(func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", TOKEN_KEY_ENDPOINT),
							ghttp.RespondWith(http.StatusOK, `{"alg":"alg", "value": "ooooppps }`),
						),
					)
				})

				It("returns the error", func() {
					Expect(err).To(HaveOccurred())
					Expect(key).To(BeEmpty())
				})

				It("logs error message", func() {
					Expect(logger).Should(gbytes.Say("error-in-unmarshaling-key"))
				})
			})

			Context("and returns an http error", func() {
				BeforeEach(func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", TOKEN_KEY_ENDPOINT),
							ghttp.RespondWith(http.StatusInternalServerError, `{}`),
						),
					)
				})

				It("returns the error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err).Should(Equal(errors.New("http-error-fetching-key")))
					Expect(key).To(BeEmpty())
				})

				It("logs error message", func() {
					Expect(logger).Should(gbytes.Say("http-error-fetching-key"))
				})
			})
		})

		Context("when UAA is unavailable", func() {

			BeforeEach(func() {
				cfg.UaaEndpoint = "http://127.0.0.1:1111"
				client, err = uaa_go_client.NewClient(logger, cfg, clock)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns the error", func() {
				Expect(err).To(HaveOccurred())
				Expect(key).To(BeEmpty())
			})

			It("logs error message", func() {
				Expect(logger).Should(gbytes.Say("error-in-fetching-key"))
			})
		})
	})
})
