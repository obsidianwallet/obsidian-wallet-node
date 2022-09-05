package api

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/obsidianwallet/obsidian-wallet-node/walletmanager"
)

func InitAPIService(address string, wlm *walletmanager.WalletManager, networkController NetworkController) (*APIService, error) {
	api := &APIService{
		address: address,
		wlm:     wlm,
	}
	return api, nil
}

func (api *APIService) Serve() error {
	log.Println("initiating api-service...")

	r := gin.Default()
	r.Use(gzip.Gzip(gzip.DefaultCompression))

	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "HEAD", "OPTIONS", "DELETE"},
		AllowHeaders:    []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		MaxAge:          12 * time.Hour,
	}))

	apiv1 := r.Group("/v1")

	apiv1.GET("/tokenlist", api.GetTokenList)

	wl := apiv1.Group("/wallet")
	wl.GET("/list_accounts", api.ListAccounts)
	wl.POST("/create_account", api.CreateAccount)
	wl.POST("/update_account", api.UpdateAccount)
	wl.GET("/delete_account", api.DeleteAccount)
	wl.GET("/get_account", api.GetAccount)
	wl.POST("/watch_token", api.WatchToken)

	pdex := apiv1.Group("/pdex")
	pdex.GET("/listpools", api.ListPools)
	pdex.GET("/listpairs", api.ListPairs)

	return r.Run(api.address)
}

func (api *APIService) GetTokenList(c *gin.Context) {

}

func (api *APIService) ListAccounts(c *gin.Context) {

}

func (api *APIService) CreateAccount(c *gin.Context) {

}

func (api *APIService) UpdateAccount(c *gin.Context) {

}

func (api *APIService) DeleteAccount(c *gin.Context) {

}

func (api *APIService) GetAccount(c *gin.Context) {

}

func (api *APIService) ListPools(c *gin.Context) {

}

func (api *APIService) ListPairs(c *gin.Context) {

}

func (api *APIService) WatchToken(c *gin.Context) {
	account := c.Query("account")
	tokenid := c.Query("tokenid")
	action := c.Query("action")
	acc := api.wlm.GetAccountInstance(account)

	if acc == nil {
		c.JSON(200, gin.H{"error": "account not found"})
		return
	}
	if action == "remove" {
		err := acc.RemoveWatchToken(tokenid)
		if err != nil {
			c.JSON(200, gin.H{"error": err})
			return
		}
		c.JSON(200, gin.H{"result": "ok"})
		return
	}
	err := acc.AddWatchToken(tokenid)
	if err != nil {
		c.JSON(200, gin.H{"error": err})
		return
	}

}
