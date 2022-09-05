package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/incognitochain/go-incognito-sdk-v2/incclient"
	"github.com/obsidianwallet/obsidian-wallet-node/api"
	"github.com/obsidianwallet/obsidian-wallet-node/common"
	"github.com/obsidianwallet/obsidian-wallet-node/database"
	"github.com/obsidianwallet/obsidian-wallet-node/pdexservice"
	"github.com/obsidianwallet/obsidian-wallet-node/walletmanager"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	err := loadConfig()
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	db, err := database.InitDatabase()
	if err != nil {
		panic(err)
	}

	netwrokController, err := NewNetworkController(cfg.UseNetwork, cfg.Networks)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	wlm, err := walletmanager.InitWallet(db)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	pdex, err := pdexservice.InitPDexService()
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	apis, err := api.InitAPIService(common.DefaultConfig.ServingAddress, wlm, netwrokController)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	netwrokController.AddNetworkUser(wlm)
	netwrokController.AddNetworkUser(apis)
	netwrokController.AddNetworkUser(pdex)

	err = netwrokController.Start()
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	err = apis.Serve()
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
}

func initChainClient(network common.NetworkID) (*incclient.IncClient, error) {
	incClient, err := incclient.NewIncClientWithCache(network.RPC, incclient.MainNetETHHost, 2, network.Name)
	if err != nil {
		return nil, err
	}
	return incClient, nil
}
