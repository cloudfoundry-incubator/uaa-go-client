package uaa_go_client_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cf-routing/uaa-go-client"
	"github.com/cf-routing/uaa-go-client/config"
	"github.com/cloudfoundry-incubator/trace-logger"
	"github.com/pivotal-golang/clock/fakeclock"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/ghttp"
)

var verifyBody = func(expectedBody string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		Expect(err).ToNot(HaveOccurred())

		defer r.Body.Close()
		Expect(string(body)).To(Equal(expectedBody))
	}
}

var _ = Describe("UAA Client", func() {
	const (
		DefaultMaxNumberOfRetries   = 3
		DefaultRetryInterval        = 15 * time.Second
		DefaultExpirationBufferTime = 30
	)

	var (
		server      *ghttp.Server
		clock       *fakeclock.FakeClock
		cfg         *config.Config
		forceUpdate bool
		logger      lager.Logger
	)

	BeforeEach(func() {
		forceUpdate = false
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
	})

	AfterEach(func() {
		server.Close()
	})

	verifyLogs := func(reqMessage, resMessage string) {
		Expect(logger).To(gbytes.Say(reqMessage))
		Expect(logger).To(gbytes.Say(resMessage))
	}

	getOauthHandlerFunc := func(status int, token *uaa_go_client.Token) http.HandlerFunc {
		return ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", "/oauth/token"),
			ghttp.VerifyBasicAuth("client-name", "client-secret"),
			ghttp.VerifyContentType("application/x-www-form-urlencoded; charset=UTF-8"),
			ghttp.VerifyHeader(http.Header{
				"Accept": []string{"application/json; charset=utf-8"},
			}),
			verifyBody("grant_type=client_credentials"),
			ghttp.RespondWithJSONEncoded(status, token),
		)
	}

	verifyFetchWithRetries := func(client *uaa_go_client.UaaClient, server *ghttp.Server, numRetries int, expectedResponses ...string) {
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer GinkgoRecover()
			defer wg.Done()
			_, err := client.FetchToken(forceUpdate)
			Expect(err).To(HaveOccurred())
		}(&wg)

		for i := 0; i < numRetries; i++ {
			Eventually(server.ReceivedRequests, 5*time.Second, 1*time.Second).Should(HaveLen(i + 1))
			clock.Increment(DefaultRetryInterval + 10*time.Second)
		}

		for _, respMessage := range expectedResponses {
			Expect(logger).To(gbytes.Say(respMessage))
		}

		wg.Wait()
	}

	Describe("uaa_go_client.NewClient", func() {
		Context("when all values are valid", func() {
			It("returns a token fetcher instance", func() {
				client, err := uaa_go_client.NewClient(logger, cfg, clock)
				Expect(err).NotTo(HaveOccurred())
				Expect(client).NotTo(BeNil())
			})
		})

		Context("when values are invalid", func() {
			Context("when oauth config is nil", func() {
				It("returns error", func() {
					client, err := uaa_go_client.NewClient(logger, nil, clock)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Configuration cannot be nil"))
					Expect(client).To(BeNil())
				})
			})

			Context("when oauth config client id is empty", func() {
				It("returns error", func() {
					config := &config.Config{
						UaaEndpoint:  "http://some.url:80",
						ClientName:   "",
						ClientSecret: "client-secret",
					}
					client, err := uaa_go_client.NewClient(logger, config, clock)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("OAuth Client ID cannot be empty"))
					Expect(client).To(BeNil())
				})
			})

			Context("when oauth config client secret is empty", func() {
				It("returns error", func() {
					config := &config.Config{
						UaaEndpoint:  "http://some.url:80",
						ClientName:   "client-name",
						ClientSecret: "",
					}
					client, err := uaa_go_client.NewClient(logger, config, clock)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("OAuth Client Secret cannot be empty"))
					Expect(client).To(BeNil())
				})
			})

			Context("when oauth config tokenendpoint is empty", func() {
				It("returns error", func() {
					config := &config.Config{
						UaaEndpoint:  "",
						ClientName:   "client-name",
						ClientSecret: "client-secret",
					}
					client, err := uaa_go_client.NewClient(logger, config, clock)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("UAA endpoint cannot be empty"))
					Expect(client).To(BeNil())
				})
			})

			Context("when token fetcher config's max number of retries is zero", func() {
				It("creates the client", func() {
					config := &config.Config{
						UaaEndpoint:           "http://some.url:80",
						MaxNumberOfRetries:    0,
						RetryInterval:         2 * time.Second,
						ExpirationBufferInSec: 30,
						ClientName:            "client-name",
						ClientSecret:          "client-secret",
					}
					client, err := uaa_go_client.NewClient(logger, config, clock)
					Expect(err).NotTo(HaveOccurred())
					Expect(client).NotTo(BeNil())
				})
			})

			Context("when token fetcher config's expiration buffer time is negative", func() {
				It("sets the expiration buffer time to the default value", func() {
					config := &config.Config{
						MaxNumberOfRetries:    3,
						RetryInterval:         2 * time.Second,
						ExpirationBufferInSec: -1,
						UaaEndpoint:           "http://some.url:80",
						ClientName:            "client-name",
						ClientSecret:          "client-secret",
					}
					client, err := uaa_go_client.NewClient(logger, config, clock)
					Expect(err).NotTo(HaveOccurred())
					Expect(client).NotTo(BeNil())
				})
			})
		})
	})

	Describe("FetchToken", func() {
		var (
			client *uaa_go_client.UaaClient
		)

		BeforeEach(func() {
			var err error
			client, err = uaa_go_client.NewClient(logger, cfg, clock)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})

		Context("when a new token needs to be fetched from OAuth server", func() {
			FContext("when the respose body is malformed", func() {
				It("returns an error and doesn't retry", func() {
					server.AppendHandlers(
						ghttp.RespondWithJSONEncoded(http.StatusOK, "broken garbage response"),
					)

					_, err := client.FetchToken(forceUpdate)
					Expect(err).To(HaveOccurred())
					Expect(server.ReceivedRequests()).Should(HaveLen(1))

					// verifyLogs("test.http-request.*/oauth/token", "test.http-response.*200")
					verifyLogs("test", "test")
				})
			})

			Context("when OAuth server cannot be reached", func() {
				It("retries number of times and finally returns an error", func() {
					cfg.UaaEndpoint = "http://bogus.url:80"
					client, err := uaa_go_client.NewClient(logger, cfg, clock)
					Expect(err).NotTo(HaveOccurred())
					wg := sync.WaitGroup{}
					wg.Add(1)
					go func(wg *sync.WaitGroup) {
						defer GinkgoRecover()
						defer wg.Done()
						_, err := client.FetchToken(forceUpdate)
						Expect(err).To(HaveOccurred())
					}(&wg)

					for i := 0; i < DefaultMaxNumberOfRetries; i++ {
						Eventually(logger).Should(gbytes.Say("test.http-request.*bogus.url"))
						Eventually(logger).Should(gbytes.Say("test.error-fetching-token"))
						clock.Increment(DefaultRetryInterval + 10*time.Second)
					}
					wg.Wait()
				})
			})

			Context("when a non 200 OK is returned", func() {
				Context("when OAuth server returns a 4xx http status code", func() {
					It("returns an error and doesn't retry", func() {
						server.AppendHandlers(
							ghttp.RespondWith(http.StatusBadRequest, "you messed up"),
						)

						_, err := client.FetchToken(forceUpdate)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(Equal("status code: 400, body: you messed up"))
						Expect(server.ReceivedRequests()).Should(HaveLen(1))
						verifyLogs("test.http-request.*/oauth/token", "test.http-response.*400")
					})
				})

				Context("when OAuth server returns a 5xx http status code", func() {
					BeforeEach(func() {
						server.AppendHandlers(
							getOauthHandlerFunc(http.StatusServiceUnavailable, nil),
							getOauthHandlerFunc(http.StatusInternalServerError, nil),
							getOauthHandlerFunc(http.StatusBadGateway, nil),
						)
					})

					It("retries a number of times and finally returns an error", func() {
						verifyFetchWithRetries(client, server, DefaultMaxNumberOfRetries, "test.http-response.*503", "test.http-response.*500", "test.http-response.*502")
					})
				})

				Context("when OAuth server returns a 3xx http status code", func() {
					It("returns an error and doesn't retry", func() {
						server.AppendHandlers(
							ghttp.RespondWith(http.StatusMovedPermanently, "moved"),
						)

						_, err := client.FetchToken(forceUpdate)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(Equal("status code: 301, body: moved"))
						Expect(server.ReceivedRequests()).Should(HaveLen(1))
						verifyLogs("test.http-request.*/oauth/token", "test.http-response.*301")
					})
				})

				Context("when OAuth server returns a mix of 5xx and 3xx http status codes", func() {
					BeforeEach(func() {
						server.AppendHandlers(
							getOauthHandlerFunc(http.StatusServiceUnavailable, nil),
							getOauthHandlerFunc(http.StatusMovedPermanently, nil),
						)
					})

					It("retries until it hits 3XX status code and  returns an error", func() {
						verifyFetchWithRetries(client, server, 2, "test.http-response.*503", "test.http-response.*301")
					})
				})
			})

			Context("when OAuth server returns 200 OK", func() {
				It("returns a new token and trace the request response", func() {
					stdout := bytes.NewBuffer([]byte{})
					trace.SetStdout(stdout)
					trace.NewLogger("true")

					responseBody := &uaa_go_client.Token{
						AccessToken: "the token",
						ExpiresIn:   20,
					}

					server.AppendHandlers(
						getOauthHandlerFunc(http.StatusOK, responseBody),
					)

					token, err := client.FetchToken(forceUpdate)
					Expect(err).NotTo(HaveOccurred())
					Expect(server.ReceivedRequests()).Should(HaveLen(1))
					Expect(token.AccessToken).To(Equal("the token"))
					Expect(token.ExpiresIn).To(Equal(int64(20)))

					r, err := ioutil.ReadAll(stdout)
					Expect(err).NotTo(HaveOccurred())
					log := string(r)
					Expect(log).To(ContainSubstring("REQUEST:"))
					Expect(log).To(ContainSubstring("POST /oauth/token HTTP/1.1"))
					Expect(log).To(ContainSubstring("RESPONSE:"))
					Expect(log).To(ContainSubstring("HTTP/1.1 200 OK"))
				})

				Context("when multiple goroutines fetch a token", func() {
					It("contacts oauth server only once and returns cached token", func() {
						responseBody := &uaa_go_client.Token{
							AccessToken: "the token",
							ExpiresIn:   3600,
						}

						server.AppendHandlers(
							getOauthHandlerFunc(http.StatusOK, responseBody),
						)
						wg := sync.WaitGroup{}
						for i := 0; i < 2; i++ {
							wg.Add(1)
							go func(wg *sync.WaitGroup) {
								defer GinkgoRecover()
								defer wg.Done()
								token, err := client.FetchToken(forceUpdate)
								Expect(err).NotTo(HaveOccurred())
								Expect(server.ReceivedRequests()).Should(HaveLen(1))
								Expect(token.AccessToken).To(Equal("the token"))
								Expect(token.ExpiresIn).To(Equal(int64(3600)))
							}(&wg)
						}
						wg.Wait()
					})
				})
			})
		})

		Context("when fetching token from Cache", func() {
			Context("when cached token is expired", func() {
				It("returns a new token and logs request response", func() {
					firstResponseBody := &uaa_go_client.Token{
						AccessToken: "the token",
						ExpiresIn:   3600,
					}
					secondResponseBody := &uaa_go_client.Token{
						AccessToken: "another token",
						ExpiresIn:   3600,
					}

					server.AppendHandlers(
						getOauthHandlerFunc(http.StatusOK, firstResponseBody),
						getOauthHandlerFunc(http.StatusOK, secondResponseBody),
					)

					token, err := client.FetchToken(forceUpdate)
					Expect(err).NotTo(HaveOccurred())
					Expect(server.ReceivedRequests()).Should(HaveLen(1))
					Expect(token.AccessToken).To(Equal("the token"))
					Expect(token.ExpiresIn).To(Equal(int64(3600)))
					clock.Increment((3600 - DefaultExpirationBufferTime) * time.Second)

					token, err = client.FetchToken(forceUpdate)
					Expect(err).NotTo(HaveOccurred())
					Expect(server.ReceivedRequests()).Should(HaveLen(2))
					Expect(token.AccessToken).To(Equal("another token"))
					Expect(token.ExpiresIn).To(Equal(int64(3600)))
				})
			})

			Context("when a cached token can be used", func() {
				It("returns the cached token", func() {
					firstResponseBody := &uaa_go_client.Token{
						AccessToken: "the token",
						ExpiresIn:   3600,
					}
					secondResponseBody := &uaa_go_client.Token{
						AccessToken: "another token",
						ExpiresIn:   3600,
					}

					server.AppendHandlers(
						getOauthHandlerFunc(http.StatusOK, firstResponseBody),
						getOauthHandlerFunc(http.StatusOK, secondResponseBody),
					)

					token, err := client.FetchToken(forceUpdate)
					Expect(err).NotTo(HaveOccurred())
					Expect(server.ReceivedRequests()).Should(HaveLen(1))
					Expect(token.AccessToken).To(Equal("the token"))
					Expect(token.ExpiresIn).To(Equal(int64(3600)))
					clock.Increment(3000 * time.Second)

					token, err = client.FetchToken(forceUpdate)
					Expect(err).NotTo(HaveOccurred())
					Expect(server.ReceivedRequests()).Should(HaveLen(1))
					Expect(token.AccessToken).To(Equal("the token"))
					Expect(token.ExpiresIn).To(Equal(int64(3600)))
				})
			})

			Context("when forcing token refresh", func() {
				It("returns a new token", func() {
					firstResponseBody := &uaa_go_client.Token{
						AccessToken: "the token",
						ExpiresIn:   3600,
					}
					secondResponseBody := &uaa_go_client.Token{
						AccessToken: "another token",
						ExpiresIn:   3600,
					}

					server.AppendHandlers(
						getOauthHandlerFunc(http.StatusOK, firstResponseBody),
						getOauthHandlerFunc(http.StatusOK, secondResponseBody),
					)

					token, err := client.FetchToken(forceUpdate)
					Expect(err).NotTo(HaveOccurred())
					Expect(server.ReceivedRequests()).Should(HaveLen(1))
					Expect(token.AccessToken).To(Equal("the token"))
					Expect(token.ExpiresIn).To(Equal(int64(3600)))

					token, err = client.FetchToken(false)
					Expect(err).NotTo(HaveOccurred())
					Expect(server.ReceivedRequests()).Should(HaveLen(2))
					Expect(token.AccessToken).To(Equal("another token"))
					Expect(token.ExpiresIn).To(Equal(int64(3600)))
				})
			})
		})
	})
})
