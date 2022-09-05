package pdexservice

import "github.com/incognitochain/go-incognito-sdk-v2/incclient"

type PDexService struct {
	serviceURL string
	incclient  *incclient.IncClient
}
