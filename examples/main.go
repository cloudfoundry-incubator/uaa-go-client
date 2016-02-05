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

	cfg := &config.Config{}
	cfg.ClientName = "client-name"
	cfg.ClientSecret = "client-secret"
	cfg.UaaEndpoint = "https://10.244.0.130"
	cfg.UseHttps = true
	cfg.SkipVerification = false

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

	fmt.Printf("Token: %#v\n", token)

}
