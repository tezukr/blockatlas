package binance

import (
	"github.com/trustwallet/blockatlas/pkg/blockatlas"
	"net/url"
)

type DexClient struct {
	blockatlas.Request
}

func (c *DexClient) GetTxsOfAddress(address, token string) (*DexTxPage, error) {
	stx := new(DexTxPage)
	query := url.Values{"address": {address}, "rows": {"25"}, "page": {"1"}, "txType": {string(TxTransfer)}}
	if token != "" {
		query.Add("txAsset", token)
	}
	err := c.Get(stx, "txs", query)
	return stx, err
}
