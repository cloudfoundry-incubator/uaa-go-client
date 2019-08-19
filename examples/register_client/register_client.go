package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	client "code.cloudfoundry.org/uaa-go-client"
	"code.cloudfoundry.org/uaa-go-client/config"
	"code.cloudfoundry.org/uaa-go-client/schema"
)

func main() {
	var (
		err       error
		uaaClient client.Client
		// token     *schema.Token
	)

	if len(os.Args) < 6 {
		fmt.Printf("Usage: <admin-client-name> <admin-client-secret> <uaa-url> <skip-verification> <new-client-id>\n\n")
		fmt.Printf("For example: admin-client-name admin-client-secret https://uaa.service.cf.internal:8443 true new-client-id\n")
		return
	}

	skip, err := strconv.ParseBool(os.Args[4])
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	cfg := &config.Config{
		ClientName:       os.Args[1],
		ClientSecret:     os.Args[2],
		UaaEndpoint:      os.Args[3],
		SkipVerification: skip,
	}

	logger := lager.NewLogger("test")
	clock := clock.NewClock()

	uaaClient, err = client.NewClient(logger, cfg, clock)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	fmt.Printf("Connecting to: %s ...\n", cfg.UaaEndpoint)

	// token, err = uaaClient.FetchToken(true)
	// if err != nil {
	// 	log.Fatal(err)
	// 	os.Exit(1)
	// }

	oauthClient := &schema.OauthClient{
		ClientId:             os.Args[5],
		Name:                 "the new client",
		ClientSecret:         "secret",
		Scope:                []string{"uaa.none"},
		ResourceIds:          []string{"none"},
		Authorities:          []string{"openid"},
		AuthorizedGrantTypes: []string{"client_credentials"},
		AccessTokenValidity:  10000,
		RedirectUri:          []string{"http://example.com"},
	}

	receivedOauthClient, err := uaaClient.RegisterOauthClient(oauthClient)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	fmt.Printf("Received Oauth Client:%+v\n", receivedOauthClient)

}
