package main

import (
	"fmt"
	"sync"

	"github.com/incognitochain/go-incognito-sdk-v2/incclient"
	"github.com/obsidianwallet/obsidian-wallet-node/common"
)

type NetworkController struct {
	currentNetwork string
	networkList    map[string]common.NetworkID
	lock           sync.Mutex
	incclient      *incclient.IncClient
	networkUsers   []NetworkUserInterface
}

type NetworkUserInterface interface {
	Stop() error
	Start() error
	SwitchNetwork(networkParam common.NetworkID, incclient *incclient.IncClient) error
}

func NewNetworkController(currentNetwork string, networkList []common.NetworkID) (*NetworkController, error) {

	nwctrl := &NetworkController{
		networkList: make(map[string]common.NetworkID),
	}

	for _, v := range networkList {
		nwctrl.networkList[v.Name] = v
	}

	if _, ok := nwctrl.networkList[currentNetwork]; !ok {
		return nil, fmt.Errorf("Network %s not found", currentNetwork)
	}

	incClient, err := initChainClient(nwctrl.networkList[currentNetwork])
	if err != nil {
		return nil, fmt.Errorf("can't use %s network, error: %v", currentNetwork, err)
	}
	nwctrl.currentNetwork = currentNetwork
	nwctrl.incclient = incClient

	return nwctrl, nil
}

func (n *NetworkController) GetNetworkList() []common.NetworkID {
	n.lock.Lock()
	defer n.lock.Unlock()
	var result []common.NetworkID
	for _, v := range n.networkList {
		result = append(result, v)
	}
	return result
}

func (n *NetworkController) GetCurrentNetwork() string {
	return n.currentNetwork
}

func (n *NetworkController) AddNetwork(networkID common.NetworkID) error {
	n.lock.Lock()
	defer n.lock.Unlock()
	if _, ok := n.networkList[networkID.Name]; !ok {
		n.networkList[networkID.Name] = networkID
	} else {
		return fmt.Errorf("network %s already exists", networkID.Name)
	}
	cfg.Networks = append(cfg.Networks, networkID)
	return updateConfigFile()
}

func (n *NetworkController) RemoveNetwork(network string) error {
	n.lock.Lock()
	defer n.lock.Unlock()
	if network == n.currentNetwork {
		return fmt.Errorf("network %s is currently in use", network)
	}
	if _, ok := n.networkList[network]; ok {
		delete(n.networkList, network)
	}
	for idx, v := range cfg.Networks {
		if v.Name == network {
			cfg.Networks = append(cfg.Networks[:idx], cfg.Networks[idx+1:]...)
			return nil
		}
	}
	return updateConfigFile()
}

func (n *NetworkController) SwitchNetwork(network string) error {
	n.lock.Lock()
	defer n.lock.Unlock()
	networkID, ok := n.networkList[network]
	if !ok {
		return fmt.Errorf("network %s not found", network)
	}
	for _, v := range n.networkUsers {
		err := v.Stop()
		if err != nil {
			return err
		}
	}
	incClient, err := initChainClient(networkID)
	if err != nil {
		return err
	}
	n.incclient = incClient
	for _, v := range n.networkUsers {
		err := v.SwitchNetwork(networkID, incClient)
		if err != nil {
			return err
		}
	}
	for _, v := range n.networkUsers {
		err := v.Start()
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *NetworkController) AddNetworkUser(networkUser NetworkUserInterface) {
	n.networkUsers = append(n.networkUsers, networkUser)
}

func (n *NetworkController) Start() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	for _, v := range n.networkUsers {
		err := v.SwitchNetwork(n.networkList[n.currentNetwork], n.incclient)
		if err != nil {
			return err
		}
	}
	for _, v := range n.networkUsers {
		err := v.Start()
		if err != nil {
			return err
		}
	}
	return nil
}
