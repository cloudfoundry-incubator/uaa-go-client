package main

import (
	"fmt"
	"log"
	"os"

	client "github.com/cf-routing/uaa-go-client"
	"github.com/cf-routing/uaa-go-client/config"
	"github.com/cf-routing/uaa-go-client/schema"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
)

func main() {
	var (
		err       error
		tlsClient *client.UaaClient
		token     *schema.Token
	)

	cfg := &config.Config{
		ClientName:       "gorouter",
		ClientSecret:     "gorouter-secret",
		UaaEndpoint:      "https://uaa.service.cf.internal:8443",
		UseHttps:         true,
		SkipVerification: true,
	}

	logger := lager.NewLogger("test")
	clock := clock.NewClock()

	tlsClient, err = client.NewClient(logger, cfg, clock)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	fmt.Printf("Connecting to: %s ...\n", cfg.UaaEndpoint)

	token, err = tlsClient.FetchToken(true)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	fmt.Printf("Response:\n\ttoken: %s\n\texpires: %d\n", token.AccessToken, token.ExpiresIn)

}
