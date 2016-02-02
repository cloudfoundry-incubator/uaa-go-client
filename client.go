package uaa_go_client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"

	trace "github.com/cloudfoundry-incubator/trace-logger"

	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"

	"github.com/cf-routing/uaa-go-client/config"
)

type Client interface {
	FetchToken(forceUpdate bool) (Token, error)
}

type UaaClient struct {
	clock            clock.Clock
	config           *config.Config
	client           *http.Client
	cachedToken      *Token
	refetchTokenTime int64
	lock             *sync.Mutex
	logger           lager.Logger
}

func NewClient(logger lager.Logger, cfg *config.Config, clock clock.Clock) (*UaaClient, error) {
	if cfg == nil {
		return nil, errors.New("Configuration cannot be nil")
	}

	if cfg.ClientName == "" {
		return nil, errors.New("OAuth Client ID cannot be empty")
	}

	if cfg.ClientSecret == "" {
		return nil, errors.New("OAuth Client Secret cannot be empty")
	}

	if cfg.UaaEndpoint == "" {
		return nil, errors.New("UAA endpoint cannot be empty")
	}

	if cfg.ExpirationBufferInSec < 0 {
		cfg.ExpirationBufferInSec = config.DefaultExpirationBufferInSec
		logger.Info("Expiration buffer in seconds set to default", lager.Data{"value": config.DefaultExpirationBufferInSec})
	}

	return &UaaClient{
		logger: logger,
		config: cfg,
		client: &http.Client{},
		clock:  clock,
		lock:   new(sync.Mutex),
	}, nil
}

func (u *UaaClient) FetchToken(forceUpdate bool) (*Token, error) {
	u.logger.Debug("fetching-token", lager.Data{"force-update": forceUpdate})
	u.lock.Lock()
	defer u.lock.Unlock()

	if !forceUpdate && u.canReturnCachedToken() {
		u.logger.Debug("return-cached-token")
		return u.cachedToken, nil
	}

	retry := true
	var retryCount uint32 = 0
	var token *Token
	var err error
	for retry == true {
		token, retry, err = u.doFetch()
		if token != nil {
			u.logger.Debug("successfully-fetched-token")
			break
		}
		if retry && retryCount < u.config.MaxNumberOfRetries {
			u.logger.Debug("retry-fetching-token", lager.Data{"retry-count": retryCount})
			retryCount++
			u.clock.Sleep(u.config.RetryInterval)
			continue
		} else {
			u.logger.Debug("failed-getting-token")
			return nil, err
		}
	}

	u.updateCachedToken(token)
	return token, nil
}

func (u *UaaClient) doFetch() (*Token, bool, error) {
	values := url.Values{}
	values.Add("grant_type", "client_credentials")
	requestBody := values.Encode()
	tokenURL := fmt.Sprintf("%s/oauth/token", u.config.UaaEndpoint)
	request, err := http.NewRequest("POST", tokenURL, bytes.NewBuffer([]byte(requestBody)))
	if err != nil {
		return nil, false, err
	}

	request.SetBasicAuth(u.config.ClientName, u.config.ClientSecret)
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	request.Header.Add("Accept", "application/json; charset=utf-8")
	trace.DumpRequest(request)
	u.logger.Debug("http-request", lager.Data{"endpoint": request.URL})

	resp, err := u.client.Do(request)
	if err != nil {
		u.logger.Debug("error-fetching-token", lager.Data{"error": err.Error()})
		return nil, true, err
	}
	defer resp.Body.Close()

	trace.DumpResponse(resp)
	u.logger.Debug("http-response", lager.Data{"status-code": resp.StatusCode})

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}

	if resp.StatusCode != http.StatusOK {
		retry := false
		if resp.StatusCode >= http.StatusInternalServerError {
			retry = true
		}
		return nil, retry, errors.New(fmt.Sprintf("status code: %d, body: %s", resp.StatusCode, body))
	}

	token := &Token{}
	err = json.Unmarshal(body, token)
	if err != nil {
		u.logger.Debug("error-umarshalling-token", lager.Data{"error": err.Error()})
		return nil, false, err
	}
	return token, false, nil
}

func (u *UaaClient) canReturnCachedToken() bool {
	return u.cachedToken != nil && u.clock.Now().Unix() < u.refetchTokenTime
}

func (u *UaaClient) updateCachedToken(token *Token) {
	u.logger.Debug("caching-token")
	u.cachedToken = token
	u.refetchTokenTime = u.clock.Now().Unix() + (token.ExpiresIn - u.config.ExpirationBufferInSec)
}
