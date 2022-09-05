package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/obsidianwallet/obsidian-wallet-node/common"
)

var cfg common.Config

func loadConfig() error {
	config, err := ioutil.ReadFile("config.json")
	if err != nil {
		return err
	}
	err = json.Unmarshal(config, &cfg)
	if err != nil {
		return err
	}
	return nil
}

func updateConfigFile() error {
	file, _ := json.MarshalIndent(cfg, "", " ")
	err := ioutil.WriteFile("config.json", file, 0644)
	if err != nil {
		return err
	}
	return nil
}
