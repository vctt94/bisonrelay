package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/companyzero/bisonrelay/client/clientdb"
	"github.com/companyzero/bisonrelay/client/clientintf"
	grpctypes "github.com/companyzero/bisonrelay/clientplugin/grpctypes"
	"github.com/companyzero/bisonrelay/zkidentity"
	"github.com/decred/dcrd/chaincfg/v3"
	"github.com/decred/slog"
	"google.golang.org/grpc"
)

type PluginClient struct {
	pluginrpc grpctypes.PluginServiceClient

	ID          string
	Name        string
	version     string
	Config      map[string]interface{}
	stream      grpctypes.PluginService_InitClient
	updatesCh   chan interface{}
	log         slog.Logger
	chainParams *chaincfg.Params
	// stream       pong.PongGame_SignalReadyClient
}

type PluginClientCfg struct {
	TLSCertPath string
	Address     string
	Log         slog.Logger
}

func (p *PluginClient) Version(ctx context.Context) (*grpctypes.PluginVersionResponse, error) {
	// Implement the logger method
	req := &grpctypes.PluginVersionRequest{}
	return p.pluginrpc.GetVersion(ctx, req)
}

func (p *PluginClient) ActionPlugin(ctx context.Context, req *grpctypes.PluginCallActionStreamRequest, cb func(grpctypes.PluginService_CallActionClient) error) error {
	stream, err := p.pluginrpc.CallAction(ctx, req)
	if err != nil {
		return err
	}

	// Invoke the callback with the stream
	if err := cb(stream); err != nil {
		return err
	}

	return nil
}

func (p *PluginClient) Logger() slog.Logger {
	// Implement the logger method

	return p.log
}

func (p *PluginClient) InitPlugin(ctx context.Context, req *grpctypes.PluginStartStreamRequest, cb func(grpctypes.PluginService_InitClient)) error {
	gameStartedStream, err := p.pluginrpc.Init(context.Background(), req)
	if err != nil {
		return fmt.Errorf("error initing stream: %w", err)
	}
	p.stream = gameStartedStream
	// pc.ID = req.ClientId
	cb(p.stream)

	return nil

}

func (p *PluginClient) ShutdownPlugin(ctx context.Context) error {
	// Implement the shutdown plugin method
	return nil
}

func NewPluginClient(ctx context.Context, cfg PluginClientCfg) (*PluginClient, error) {
	// First attempt to establish a connection to lnd's RPC sever.
	// _, err := credentials.NewClientTLSFromFile(cfg.TLSCertPath, "")
	// if err != nil {
	// 	fmt.Printf("cfg Address: %+v\n\n", cfg.Address)
	// 	return nil, fmt.Errorf("unable to read cert file: %v", err)
	// }
	// opts := []grpc.DialOption{grpc.WithTransportCredentials(creds)}

	conn, err := grpc.Dial(cfg.Address, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("unable to dial to dcrlnd's gRPC server: %v", err)
	}

	// // Start RPCs.
	pc := grpctypes.NewPluginServiceClient(conn)

	log := slog.Disabled
	if cfg.Log != nil {
		log = cfg.Log
	}

	return &PluginClient{
		updatesCh: make(chan interface{}),
		pluginrpc: pc,
		log:       log,
	}, nil
}

// InstallPlugin installs a new plugin.
func (c *Client) InstallPlugin(plugin *PluginClient) error {
	// Save plugin data to the database.
	err := c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		return c.db.SavePlugin(tx, clientdb.PluginData{
			Name: plugin.Name,
			// Version: plugin.Version,
			// Enabled:   plugin.Enabled,
			// Config:    plugin.Config,
			Installed: time.Now(),
			Updated:   time.Now(),
		})
	})
	if err != nil {
		return fmt.Errorf("failed to save plugin: %v", err)
	}

	// Add plugin to the client's plugin list.
	c.plugins[plugin.ID] = plugin
	return nil
}

// RegisterPlugin registers a new plugin with the client.
func (c *Client) RegisterPluginStream(id zkidentity.ShortID, stream grpctypes.PluginService_InitServer) {
	// c.plugins[id] = stream
}

func (c *Client) Init(ctx context.Context, req *grpctypes.PluginStartStreamRequest, cb func(clientintf.Plugin_InitClient)) error {

	// id, _ := zkidentity.Byte2ID(req.ClientId)
	// clientStream, err := c.plugins[id].InitPlugin(context.Background(), req)
	// if err != nil {
	// 	return fmt.Errorf("error creating game started stream: %w", err)
	// }
	// pc.clientStream = clientStream
	// pc.ID = req.ClientId
	// cb(pc.clientStream)

	return nil
}

// initializePlugins initializes all registered plugins.
func (c *Client) initializePlugins() error {
	// for _, plugin := range c.plugins {
	// 	if err := plugin.InitPlugin(c.ctx); err != nil {
	// 		c.log.Errorf("Failed to initialize plugin %s: %v", plugin.ID(), err)
	// 		return err
	// 	}
	// 	c.log.Infof("Initialized plugin %s", plugin.ID())
	// }
	return nil
}

// EnablePlugin enables a plugin by name.
func (c *Client) EnablePlugin(id string) error {
	plugin, exists := c.plugins[id]
	if !exists {
		return errors.New("plugin not found")
	}

	// plugin.Enabled = true
	return c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		return c.db.UpdatePluginConfig(tx, plugin.ID, plugin.Config)
	})
}

// DisablePlugin disables a plugin by name.
func (c *Client) DisablePlugin(id string) error {
	plugin, exists := c.plugins[id]
	if !exists {
		return errors.New("plugin not found")
	}

	// plugin.Enabled = false
	return c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		return c.db.UpdatePluginConfig(tx, plugin.ID, plugin.Config)
	})
}

// UpdatePluginConfig updates the configuration of a plugin.
func (c *Client) UpdatePluginConfig(id string, config map[string]interface{}) error {
	plugin, exists := c.plugins[id]
	if !exists {
		return errors.New("plugin not found")
	}

	// plugin.Config = config
	return c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		return c.db.UpdatePluginConfig(tx, plugin.ID, config)
	})
}

// LoadPlugins loads all plugins from the database into the client.
func (c *Client) LoadPlugins() error {
	// var plugins []clientdb.PluginData
	// err := c.dbView(func(tx clientdb.ReadTx) error {
	// 	var err error
	// 	plugins, err = c.db.ListPlugins(tx)
	// 	return err
	// })
	// if err != nil {
	// 	return fmt.Errorf("failed to load plugins: %v", err)
	// }

	c.plugins = make(map[string]*PluginClient)
	// for _, p := range plugins {

	// 	c.plugins[p.ID] = &Plugin{
	// 		id:      p.ID,
	// 		name:    p.Name,
	// 		version: p.Version,
	// 		enabled: p.Enabled,
	// 		config:  p.Config,
	// 	}
	// }
	return nil
}

// PerformPluginAction performs an action defined by a plugin.
func (c *Client) PerformPluginAction(pluginID zkidentity.ShortID, action string, data json.RawMessage) error {
	// plugin, exists := c.plugins[pluginID]
	// if !exists {
	// 	return errors.New("plugin not found")
	// }
	// if !plugin.(*Plugin).enabled {
	// 	return fmt.Errorf("plugin %s is not enabled", pluginID)
	// }

	// Perform the action based on the plugin's requirements.
	// This is a placeholder and should be replaced with actual plugin action logic.
	fmt.Printf("Performing action %s for plugin %s with data %s\n", action, pluginID, string(data))

	return nil
}
