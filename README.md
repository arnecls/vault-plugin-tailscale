# vault-plugin-tailscale

[![Go Reference](https://pkg.go.dev/badge/github.com/davidsbond/vault-plugin-tailscale.svg)](https://pkg.go.dev/github.com/davidsbond/vault-plugin-tailscale)
[![Go Report Card](https://goreportcard.com/badge/github.com/davidsbond/vault-plugin-tailscale)](https://goreportcard.com/report/github.com/davidsbond/vault-plugin-tailscale)
![Github Actions](https://github.com/davidsbond/vault-plugin-tailscale/actions/workflows/ci.yml/badge.svg?branch=master)

A [HashiCorp Vault](https://www.vaultproject.io/) plugin for generating device authentication keys for 
[Tailscale](https://tailscale.com). Generated keys are single use.

## Installation

1. Download the binary for your architecture from the [releases](https://github.com/davidsbond/vault-plugin-tailscale/releases) page
2. Generate the SHA256 sum of the plugin binary

```bash
SHASUM=$(sha256sum vault-plugin-tailscale | cut -d ' ' -f1)
```

3. Add the plugin to your Vault plugin catalog (requires VAULT_TOKEN to be set)

```bash
vault plugin register -sha256="${SHASUM}" secret vault-plugin-tailscale
```

4. Mount the plugin

```bash
vault secrets enable -path=tailscale vault-plugin-tailscale 
```

## Configuration

1. The ID of your tailnet is displayed on the top left of your admin console (your org name)
2. Obtain an API key or Oauth client credentials ("devices" scope) from the Tailscale admin dashboard.
3. Create the Vault configuration for the Tailscale API
   

```bash
# Authenticate through an API Key
vault write tailscale/config \
tailnet="${TAILNET}" \
api_key="${API_KEY}"
```

```bash
# Or use oauth client credentials
# Make sure to change the api_url!
vault write tailscale/config \
tailnet="${TAILNET}" \
oauth_client_id="${OAUTH_CLIENT_ID}" \
oauth_client_secret="${OAUTH_CLIENT_SECRET}" \
api_url='https://api.tailscale.com/api/v2/oauth/token'
```

## Usage

Generate keys using the Vault CLI.

```bash
vault read tailscale/key
```

This will yield the following output:

```
Key          Value
---          -----
ephemeral    false
expires      2024-08-30T00:00:00Z
id           kMxzN47CNTRL
key          ....
reusable     false
tags         
```

### Key Options

The following key/value pairs can be added to the end of the `vault read` command to configure key properties:

#### Tags

A comma separated list of tags to apply to the device that uses the authentication key.
Keys _must_ have a tag set. You can assign default tags to an oauth client on credential creation though.

```bash
vault read tailscale/key tags='tag:foo,tag:bar'
```

#### Preauthorized

If true, machines added to the tailnet with this key will not required authorization.

```bash
vault read tailscale/key preauthorized=true
```

#### Ephemeral

If true, nodes created with this key will be removed after a period of inactivity or when they disconnect from the Tailnet.

```bash
vault read tailscale/key ephemeral=true
```

#### Reusable

If true, the key can be reused for different nodes/devices.
This is useful if you're dealing with ephemeral VMs or pods.

```bash
vault read tailscale/key reusable=true
```

#### lifetime

By default the lifetime of a generated key is `90d`. You can set a shorter liftime if needed.
Durations can be set using the standard [golang duration notation](https://pkg.go.dev/maze.io/x/duration#ParseDuration).

```bash
vault read tailscale/key lifetime='24h'
```