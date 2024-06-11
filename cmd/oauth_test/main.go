package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/tailscale/tailscale-client-go/tailscale"
)

// We're testing the OAuth client credentials flow.
// The client id and client secret are being passed as commandline arguments.
//
// Arguments are given in the following order:
// 1. Organization ID
// 2. A tag to assign to the key
// 3. Oauth Client ID
// 4. Oauth Client Secret
func main() {
	org := os.Args[1]
	tag := os.Args[2]
	clientID := os.Args[3]
	clientSecret := os.Args[4]

	clientOpts := []tailscale.ClientOption{
		tailscale.WithBaseURL("https://api.tailscale.com"),
		tailscale.WithOAuthClientCredentials(clientID, clientSecret, []string{"devices"}),
	}

	client, err := tailscale.NewClient("", org, clientOpts...)
	if err != nil {
		panic(err)
	}

	var capabilities tailscale.KeyCapabilities
	capabilities.Devices.Create.Tags = strings.Split(tag, ",")
	capabilities.Devices.Create.Preauthorized = false
	capabilities.Devices.Create.Ephemeral = true

	key, err := client.CreateKey(context.Background(), capabilities)
	if err != nil {
		panic(err)
	}

	fmt.Println(key)
}
