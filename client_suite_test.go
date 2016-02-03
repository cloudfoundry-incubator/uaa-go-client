package uaa_go_client_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"

	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/cf-routing/uaa-go-client"
	"github.com/cf-routing/uaa-go-client/config"
	"github.com/cf-routing/uaa-go-client/schema"
	"github.com/pivotal-golang/clock/fakeclock"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/ghttp"
	"github.com/pivotal-golang/lager"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Client Suite")
}

const (
	DefaultMaxNumberOfRetries   = 3
	DefaultRetryInterval        = 15 * time.Second
	DefaultExpirationBufferTime = 30
)

var (
	logger      lager.Logger
	forceUpdate bool
	server      *ghttp.Server
	clock       *fakeclock.FakeClock
	cfg         *config.Config
)

var verifyBody = func(expectedBody string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		Expect(err).ToNot(HaveOccurred())

		defer r.Body.Close()
		Expect(string(body)).To(Equal(expectedBody))
	}
}

var verifyLogs = func(reqMessage, resMessage string) {
	Expect(logger).To(gbytes.Say(reqMessage))
	Expect(logger).To(gbytes.Say(resMessage))
}

var getOauthHandlerFunc = func(status int, token *schema.Token) http.HandlerFunc {
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

var verifyFetchWithRetries = func(client *uaa_go_client.UaaClient, server *ghttp.Server, numRetries int, expectedResponses ...string) {
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
