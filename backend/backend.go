// Package backend provides the Vault plugin backend that is used to generate authentication keys for Tailscale
// devices.
package backend

import (
	"context"
	"errors"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/tailscale/tailscale-client-go/tailscale"
)

// PluginVersion is set via "-X 'backend.PluginVersion=x.y.z'" during the build process.
var PluginVersion = "v0.0.0"

type (
	// The Backend type is responsible for handling inbound requests from Vault to serve Tailscale authentication
	// keys.
	Backend struct {
		*framework.Backend
	}

	// The Config type describes the configuration fields used by the Backend
	Config struct {
		Tailnet           string   `json:"tailnet"`
		APIKey            string   `json:"api_key"`
		APIUrl            string   `json:"api_url"`
		OAuthClientID     string   `json:"oauth_client_id"`
		OAuthClientSecret string   `json:"oauth_client_secret"`
		OAuthScopes       []string `json:"oauth_scopes"`
	}
)

const (
	backendHelp              = "The Tailscale backend is used to generate Tailscale authentication keys for a configured Tailnet"
	readKeyDescription       = "Generate a single-use authentication key for a device"
	readConfigDescription    = "Read the current Tailscale backend configuration"
	updateConfigDescription  = "Update the Tailscale backend configuration"
	apiKeyDescription        = "The API key to use for authenticating with the Tailscale API. Ignored if OAuth credentials are provided."
	tailnetDescription       = "The name of the Tailscale Tailnet"
	tagsDescription          = "Tags to apply to the device that uses the authentication key"
	preauthorizedDescription = "If true, machines added to the tailnet with this key will not required authorization"
	apiUrlDescription        = "The URL of the Tailscale API"
	ephemeralDescription     = "If true, nodes created with this key will be removed after a period of inactivity or when they disconnect from the Tailnet"
	oauthClientDescription   = "The OAuth client ID to use for authenticating with the Tailscale API."
	oauthSecretDescription   = "The OAuth client secret to use for authenticating with the Tailscale API."
	oauthScopesDescription   = "A comma separated list of OAuth scopes to request when authenticating with the Tailscale API. Must match the scopes configured for the used credentials"
	reusableDescription      = "If true, the key can be used for multiple, different devices"
	lifetimeSecDescription   = "The key lifetime in seconds or as a valid go duration string (>= 1s). Defaults to 90 days"
)

// Create a new logical.Backend implementation that can generate authentication keys for Tailscale devices.
func Create(ctx context.Context, config *logical.BackendConfig) (logical.Backend, error) {
	backend := &Backend{}
	backend.Backend = &framework.Backend{
		BackendType:    logical.TypeLogical,
		RunningVersion: PluginVersion,
		Help:           backendHelp,
		Paths: []*framework.Path{
			{
				Pattern: "key",
				Fields: map[string]*framework.FieldSchema{
					"tags": {
						Type:        framework.TypeCommaStringSlice,
						Description: tagsDescription,
					},
					"preauthorized": {
						Type:        framework.TypeBool,
						Description: preauthorizedDescription,
						Default:     false,
					},
					"ephemeral": {
						Type:        framework.TypeBool,
						Description: ephemeralDescription,
						Default:     false,
					},
					"reusable": {
						Type:        framework.TypeBool,
						Description: reusableDescription,
						Default:     false,
					},
					"lifetime": {
						Type:        framework.TypeDurationSecond,
						Description: lifetimeSecDescription,
						Default:     "90d",
					},
				},
				Operations: map[logical.Operation]framework.OperationHandler{
					logical.ReadOperation: &framework.PathOperation{
						Summary:  readKeyDescription,
						Callback: backend.GenerateKey,
					},
				},
			},
			{
				Pattern: "config",
				Fields: map[string]*framework.FieldSchema{
					"api_key": {
						Type:        framework.TypeString,
						Description: apiKeyDescription,
						Default:     "",
					},
					"tailnet": {
						Type:        framework.TypeString,
						Description: tailnetDescription,
					},
					"api_url": {
						Type:        framework.TypeString,
						Description: apiUrlDescription,
						Default:     "https://api.tailscale.com",
					},
					"oauth_client_id": {
						Type:        framework.TypeString,
						Description: oauthClientDescription,
						Default:     "",
					},
					"oauth_client_secret": {
						Type:        framework.TypeString,
						Description: oauthSecretDescription,
						Default:     "",
					},
					"oauth_scopes": {
						Type:        framework.TypeCommaStringSlice,
						Description: oauthScopesDescription,
						Default:     []string{"devices"},
					},
				},
				Operations: map[logical.Operation]framework.OperationHandler{
					logical.ReadOperation: &framework.PathOperation{
						Callback: backend.ReadConfiguration,
						Summary:  readConfigDescription,
					},
					logical.UpdateOperation: &framework.PathOperation{
						Callback: backend.UpdateConfiguration,
						Summary:  updateConfigDescription,
					},
				},
			},
		},
	}

	return backend, backend.Setup(ctx, config)
}

const (
	configPath = "config"
)

// GenerateKey generates a new authentication key via the Tailscale API. This method checks the existing Backend configuration
// for the Tailnet and API key. It will return an error if the configuration does not exist.
func (b *Backend) GenerateKey(ctx context.Context, request *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	entry, err := request.Storage.Get(ctx, configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err = entry.DecodeJSON(&config); err != nil {
		return nil, err
	}

	clientOpts := []tailscale.ClientOption{
		tailscale.WithBaseURL(config.APIUrl),
	}

	if config.OAuthClientID != "" {
		b.Logger().Debug("Using oauth client credentials for authentication")
		clientOpts = append(clientOpts, tailscale.WithOAuthClientCredentials(config.OAuthClientID, config.OAuthClientSecret, config.OAuthScopes))
	} else {
		b.Logger().Debug("Using auth-key for authentication")
	}

	client, err := tailscale.NewClient(config.APIKey, config.Tailnet, clientOpts...)
	if err != nil {
		return nil, err
	}

	var capabilities tailscale.KeyCapabilities
	capabilities.Devices.Create.Tags = data.Get("tags").([]string)
	capabilities.Devices.Create.Preauthorized = data.Get("preauthorized").(bool)
	capabilities.Devices.Create.Ephemeral = data.Get("ephemeral").(bool)
	capabilities.Devices.Create.Reusable = data.Get("reusable").(bool)

	lifetimeSec, isLifetimeSet, err := data.GetOkErr("lifetime")

	if !isLifetimeSet {
		lifetimeSec = data.GetDefaultOrZero("lifetime")
	} else if err != nil {
		return nil, err // likely a parsing error
	}

	lifetime := time.Duration(lifetimeSec.(int)) * time.Second

	key, err := client.CreateKey(ctx, capabilities, tailscale.WithKeyExpiry(lifetime))
	if err != nil {
		return nil, err
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"id":            key.ID,
			"key":           key.Key,
			"expires":       key.Expires,
			"tags":          key.Capabilities.Devices.Create.Tags,
			"reusable":      key.Capabilities.Devices.Create.Reusable,
			"ephemeral":     key.Capabilities.Devices.Create.Ephemeral,
			"preauthorized": key.Capabilities.Devices.Create.Preauthorized,
		},
	}, nil
}

// ReadConfiguration reads the Backend configuration and returns its values.
func (b *Backend) ReadConfiguration(ctx context.Context, request *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	entry, err := request.Storage.Get(ctx, configPath)
	switch {
	case err != nil:
		return nil, err
	case entry == nil:
		return nil, errors.New("configuration has not been set")
	}

	var config Config
	if err = entry.DecodeJSON(&config); err != nil {
		return nil, err
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"tailnet":             config.Tailnet,
			"api_key":             config.APIKey,
			"api_url":             config.APIUrl,
			"oauth_client_id":     config.OAuthClientID,
			"oauth_client_secret": config.OAuthClientSecret,
			"oauth_scopes":        config.OAuthScopes,
		},
	}, nil
}

// UpdateConfiguration modifies the Backend configuration. Returns an error if any required fields are missing.
func (b *Backend) UpdateConfiguration(ctx context.Context, request *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	config := Config{
		Tailnet:           data.Get("tailnet").(string),
		APIKey:            data.Get("api_key").(string),
		APIUrl:            data.Get("api_url").(string),
		OAuthClientID:     data.Get("oauth_client_id").(string),
		OAuthClientSecret: data.Get("oauth_client_secret").(string),
		OAuthScopes:       data.Get("oauth_scopes").([]string),
	}

	switch {
	case config.Tailnet == "":
		return nil, errors.New("provided tailnet cannot be empty")
	case config.APIKey == "" && config.OAuthClientID == "" && config.OAuthClientSecret == "":
		return nil, errors.New("must either provide a non-empty api_key or a non-empty oauth_client_id and oauth_client_secret")
	case config.APIUrl == "":
		return nil, errors.New("provided api_url cannot be empty")
	}

	entry, err := logical.StorageEntryJSON(configPath, config)
	if err != nil {
		return nil, err
	}

	if err = request.Storage.Put(ctx, entry); err != nil {
		return nil, err
	}

	return &logical.Response{}, nil
}
