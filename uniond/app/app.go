package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	"github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/cosmos-sdk/x/staking/keeper"
)

const (
	AppName              = "union"
	DefaultAccountPrefix = "union"
)

var DefaultNodeHome string

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	DefaultNodeHome = filepath.Join(userHomeDir, "."+AppName)
}

// UnionApp represents the main blockchain application.
type UnionApp struct {
	*baseapp.BaseApp
	logger      types.Logger
	appCodec    codec.Codec
	keys        map[string]*types.KVStoreKey
	AuthKeeper  keeper.AccountKeeper
	BankKeeper  keeper.BaseKeeper
	StakingKeeper keeper.Keeper
	ModuleManager *module.Manager
}

// NewUnionApp creates a new UnionApp instance.
func NewUnionApp(
	logger types.Logger,
	db types.KVStore,
	traceStore io.Writer,
	appOpts servertypes.AppOptions,
) *UnionApp {
	appCodec := codec.NewProtoCodec(types.NewInterfaceRegistry())
	keys := make(map[string]*types.KVStoreKey)
	app := &UnionApp{
		BaseApp: baseapp.NewBaseApp(AppName, logger, db, nil),
		logger:  logger,
		appCodec: appCodec,
		keys:     keys,
	}

	app.initKeepers()
	app.initModuleManager()
	app.SetInitChainer(app.initChainer)
	app.SetBeginBlocker(app.beginBlocker)
	app.SetEndBlocker(app.endBlocker)

	if err := app.LoadLatestVersion(); err != nil {
		panic(fmt.Errorf("failed to load latest version: %w", err))
	}

	return app
}

func (app *UnionApp) initKeepers() {
	app.AuthKeeper = keeper.NewAccountKeeper(
		app.appCodec,
		app.keys[auth.StoreKey],
		DefaultAccountPrefix,
	)

	app.BankKeeper = keeper.NewBaseKeeper(
		app.appCodec,
		app.keys[bank.StoreKey],
		app.AuthKeeper,
	)

	app.StakingKeeper = keeper.NewKeeper(
		app.appCodec,
		app.keys[staking.StoreKey],
		app.AuthKeeper,
		app.BankKeeper,
	)
}

func (app *UnionApp) initModuleManager() {
	app.ModuleManager = module.NewManager(
		auth.NewAppModule(app.appCodec, app.AuthKeeper),
		bank.NewAppModule(app.appCodec, app.BankKeeper),
		staking.NewAppModule(app.appCodec, app.StakingKeeper),
	)
}

func (app *UnionApp) initChainer(ctx types.Context, req *abci.InitChainRequest) (*abci.InitChainResponse, error) {
	var genesisState map[string]json.RawMessage
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		return nil, err
	}
	return app.ModuleManager.InitGenesis(ctx, genesisState)
}

func (app *UnionApp) beginBlocker(ctx types.Context) (types.BeginBlock, error) {
	return app.ModuleManager.BeginBlock(ctx)
}

func (app *UnionApp) endBlocker(ctx types.Context) (types.EndBlock, error) {
	return app.ModuleManager.EndBlock(ctx)
}

func (app *UnionApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx
	app.ModuleManager.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
}
