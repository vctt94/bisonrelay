package client

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/companyzero/bisonrelay/client/clientdb"
	"github.com/companyzero/bisonrelay/client/clientintf"
	"github.com/companyzero/bisonrelay/clientrpc/types"
	"github.com/decred/slog"
)

// PluginClient manages the connection and communication with a plugin server.
type PluginClient struct {
	pluginrpc types.PluginServiceClient

	ID        clientdb.PluginID `json:"id"`
	Name      string            `json:"name"`
	Version   string            `json:"version"`
	Config    PluginClientCfg   `json:"config"`
	Enabled   bool              `json:"enabled"`
	Installed time.Time
	Updated   time.Time

	UpdateCh chan *types.PluginCallActionStreamResponse
	NtfnCh   chan *types.PluginStartStreamResponse

	stream types.PluginService_InitClient
	log    slog.Logger
}

// PluginClientCfg holds the configuration needed to connect to a plugin server.
type PluginClientCfg struct {
	TLSCertPath string
	Address     string
	Log         slog.Logger
}

// NewPluginClient initializes a new PluginClient with the given configuration and context.
func NewPluginClient(ctx context.Context, id clientdb.PluginID, cfg PluginClientCfg) (*PluginClient, error) {
	// Load the server's certificate for TLS encryption.

	// Set up the logger; use a disabled logger if none is provided.
	log := slog.Disabled
	if cfg.Log != nil {
		log = cfg.Log
	}

	// Initialize the PluginClient.
	p := &PluginClient{
		ID:  id,
		log: log,
		Config: PluginClientCfg{
			Address:     cfg.Address,
			TLSCertPath: cfg.TLSCertPath,
		},
		Enabled:  true,
		UpdateCh: make(chan *types.PluginCallActionStreamResponse),
		NtfnCh:   make(chan *types.PluginStartStreamResponse),
	}

	return p, nil
}

// GetVersion retrieves the version information of the plugin from the server.
func (p *PluginClient) GetVersion(ctx context.Context, req *types.PluginVersionRequest, resp *types.PluginVersionResponse) error {
	return p.pluginrpc.GetVersion(ctx, req, resp)
}

// CallPluginAction initiates an action on the plugin and processes the stream with a callback.
func (p *PluginClient) CallPluginAction(ctx context.Context, req *types.PluginCallActionStreamRequest, cb func(types.PluginService_CallActionClient) error) error {
	stream, err := p.pluginrpc.CallAction(ctx, req)
	if err != nil {
		return err
	}

	// Invoke the callback with the stream.
	if err := cb(stream); err != nil {
		return err
	}

	return nil
}

func (p *PluginClient) CallPluginInput(ctx context.Context, req *types.PluginInputRequest, resp *types.PluginInputResponse) error {
	return p.pluginrpc.SendInput(ctx, req, resp)
}

// Render processes data received from the plugin server for display or further use.
func (p *PluginClient) Render(ctx context.Context, data *types.PluginCallActionStreamResponse, resp *types.RenderResponse) error {
	req := &types.RenderRequest{
		Data: data.Response,
	}
	return p.pluginrpc.Render(ctx, req, resp)
}

// Logger returns the logger associated with the PluginClient.
func (p *PluginClient) Logger() slog.Logger {
	return p.log
}

// InitPlugin initializes the plugin on the server and sets up a notification stream.
func (p *PluginClient) InitPlugin(ctx context.Context, req *types.PluginStartStreamRequest, cb func(types.PluginService_InitClient)) error {
	// Start the plugin stream.
	startedStream, err := p.pluginrpc.Init(context.Background(), req)
	if err != nil {
		return fmt.Errorf("error initializing stream: %w", err)
	}
	p.stream = startedStream

	// Invoke the callback with the initialized stream.
	cb(p.stream)

	return nil
}

// SavePluginInfo saves the plugin's configuration and details to the database.
func (c *Client) SavePluginInfo(plugin *PluginClient) error {
	// Ensure the plugin does not already exist.
	return c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		_, err := c.db.GetPlugin(tx, plugin.ID)
		if err == nil {
			return fmt.Errorf("plugin %s already exists: %w", plugin.Name, clientdb.ErrAlreadyExists)
		} else if !errors.Is(err, clientdb.ErrNotFound) {
			return err
		}

		// Convert PluginClientCfg to a map for database storage.
		config := map[string]interface{}{
			"address":     plugin.Config.Address,
			"tlsCertPath": plugin.Config.TLSCertPath,
		}

		// Create a database record for the plugin.
		pdb := clientdb.Plugin{
			ID:        plugin.ID.String(),
			Name:      plugin.Name,
			Version:   plugin.Version,
			Config:    config,
			Enabled:   plugin.Enabled,
			Installed: time.Now(),
		}

		// Save the plugin data to the database.
		return c.db.SavePlugin(tx, pdb)
	})
}

// ListPlugins retrieves all plugins saved in the database and returns them as PluginClient.
func (c *Client) ListPlugins() ([]PluginClient, error) {
	var res []PluginClient
	err := c.dbView(func(tx clientdb.ReadTx) error {
		plugins, err := c.db.ListPlugins(tx)
		if err != nil {
			return err
		}

		// Convert all plugins to PluginClient.
		for _, plugin := range plugins {
			// Retrieve plugin configuration details.
			address, ok := plugin.Config["address"].(string)
			if !ok {
				return fmt.Errorf("address not found in plugin config for %s", plugin.ID)
			}
			tlsCertPath, ok := plugin.Config["tlsCertPath"].(string)
			if !ok {
				return fmt.Errorf("TLS certificate path not found in plugin config for %s", plugin.ID)
			}

			// Create a PluginClient from the plugin configuration.
			pc := PluginClient{
				ID:      UserIDFromStr(plugin.ID),
				Name:    plugin.Name,
				Version: plugin.Version,
				Config: PluginClientCfg{
					Address:     address,
					TLSCertPath: tlsCertPath,
				},
				Enabled: plugin.Enabled,
			}
			res = append(res, pc)
		}
		return nil
	})

	return res, err
}

// GetEnabledPlugins returns a list of all enabled plugins from the database.
func (c *Client) GetEnabledPlugins() ([]PluginClient, error) {
	var res []PluginClient
	err := c.dbView(func(tx clientdb.ReadTx) error {
		plugins, err := c.db.ListPlugins(tx)
		if err != nil {
			return err
		}

		// Filter enabled plugins and convert to PluginClient.
		for _, plugin := range plugins {
			if plugin.Enabled {
				// Retrieve plugin configuration details.
				address, ok := plugin.Config["address"].(string)
				if !ok {
					return fmt.Errorf("address not found in plugin config for %s", plugin.ID)
				}
				tlsCertPath, ok := plugin.Config["tlsCertPath"].(string)
				if !ok {
					return fmt.Errorf("TLS certificate path not found in plugin config for %s", plugin.ID)
				}

				// Create a PluginClient from the plugin configuration.
				pc := PluginClient{
					ID:      UserIDFromStr(plugin.ID),
					Name:    plugin.Name,
					Version: plugin.Version,
					Config: PluginClientCfg{
						Address:     address,
						TLSCertPath: tlsCertPath,
					},
					Enabled: plugin.Enabled,
				}
				res = append(res, pc)
			}
		}
		return nil
	})

	return res, err
}

// AddNewPlugin initializes and adds a new plugin to the client.
func (c *Client) AddNewPlugin(ctx context.Context, pid clientintf.PluginID, address, tlsCertPath string) (*PluginClient, error) {
	// Create a new PluginClient configuration.
	cfg := PluginClientCfg{
		Address:     address,
		TLSCertPath: tlsCertPath,
		Log:         c.log,
	}

	// Initialize a new PluginClient.
	pluginClient, err := NewPluginClient(ctx, pid, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize plugin: %w", err)
	}

	// Save the plugin information in the database.
	err = c.SavePluginInfo(pluginClient)
	if err != nil {
		return nil, fmt.Errorf("failed to save plugin info: %w", err)
	}

	return pluginClient, nil
}
