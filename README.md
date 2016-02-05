[![Build Status](https://travis-ci.org/cf-routing/uaa-go-client.svg?branch=master)](https://travis-ci.org/cf-routing/uaa-go-client)

# uaa-go-client
UAA Client for Go!

# Example (non-TLS)

```
cfg := &config.Config{
	ClientName:       "gorouter",
	ClientSecret:     "gorouter-secret",
	UaaEndpoint:      "http://uaa.service.cf.internal",
}


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
```


# Example (TLS)

```
cfg := &config.Config{
	ClientName:       "gorouter",
	ClientSecret:     "gorouter-secret",
	UaaEndpoint:      "https://uaa.service.cf.internal:8443",
	UseHttps:         true,
	SkipVerification: true,
}

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
```
