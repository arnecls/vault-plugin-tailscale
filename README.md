# vault-plugin-tailscale

[![Go Reference](https://pkg.go.dev/badge/github.com/davidsbond/vault-plugin-tailscale.svg)](https://pkg.go.dev/github.com/davidsbond/vault-plugin-tailscale)
[![Go Report Card](https://goreportcard.com/badge/github.com/davidsbond/vault-plugin-tailscale)](https://goreportcard.com/report/github.com/davidsbond/vault-plugin-tailscale)
![Github Actions](https://github.com/davidsbond/vault-plugin-tailscale/actions/workflows/ci.yml/badge.svg?branch=master)

A [HashiCorp Vault](https://www.vaultproject.io/) plugin for generating device authentication keys for 
[Tailscale](https://tailscale.com). Generated keys are single use.

## Installation

1. Download the binary for your architecture from the [releases](https://github.com/davidsbond/vault-plugin-tailscale/releases) page
2. Generate the SHA256 sum of the plugin binary

```shell
SHASUM=$(sha256sum vault-plugin-tailscale | cut -d ' ' -f1)
```

3. Add the plugin to your Vault plugin catalog (requires VAULT_TOKEN to be set)

```shell
vault plugin register -sha256="${SHASUM}" secret vault-plugin-tailscale
```

4. Mount the plugin

```shell
vault secrets enable -path=tailscale vault-plugin-tailscale 
```

## Configuration

1. The ID of your tailnet is displayed on the top left of your admin console (your org name)
2. Obtain an API key or Oauth client credentials ("all" scope) from the Tailscale admin dashboard.
3. Create the Vault configuration for the Tailscale API
   

```shell
# Authenticate through an API Keu
vault write tailscale/config \
tailnet="${TAILNET}" \
api_key="${API_KEY}"
```

```shell
# Or use oauth client credentials
# Make sure to change the api_url!
vault write tailscale/config tailnet="${TAILNET}" \
oauth_client_id="${OAUTH_CLIENT_ID}" \
oauth_client_secret="${OAUTH_CLIENT_SECRET}" \
api_url='https://api.tailscale.com/api/v2/oauth/token'
```

## Usage

Generate keys using the Vault CLI.

```shell
$ vault read tailscale/key
Key          Value
---          -----
ephemeral    false
expires      2022-04-30T00:32:36Z
id           kMxzN47CNTRL
key          secret-key-data
reusable     false
tags         <nil>
```

### Key Options

The following key/value pairs can be added to the end of the `vault read` command to configure key properties:

#### Tags

Tags to apply to the device that uses the authentication key

```shell
vault read tailscale/key tags=something:somewhere
```

#### Preauthorized

If true, machines added to the tailnet with this key will not required authorization

```shell
vault read tailscale/key preauthorized=true
```

#### Ephemeral

If true, nodes created with this key will be removed after a period of inactivity or when they disconnect from the Tailnet

```shell
vault read tailscale/key ephemeral=true
```
