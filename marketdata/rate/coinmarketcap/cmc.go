package cmc

import (
	"github.com/spf13/viper"
	"github.com/trustwallet/blockatlas/marketdata/cmcmap"
	"github.com/trustwallet/blockatlas/marketdata/rate"
	"github.com/trustwallet/blockatlas/pkg/blockatlas"
	"math/big"
	"net/url"
)

type Cmc struct {
	rate.Rate
}

func InitRate() rate.Provider {
	cmc := &Cmc{
		Rate: rate.Rate{
			Id:         "cmc",
			Request:    blockatlas.InitClient(viper.GetString("market.cmc.api")),
			UpdateTime: viper.GetString("market.cmc.rate_update_time"),
		},
	}
	cmc.Headers["X-CMC_PRO_API_KEY"] = viper.GetString("market.cmc.api_key")
	return cmc
}

func (c *Cmc) FetchLatestRates() (rates blockatlas.Rates, err error) {
	cmap, err := cmcmap.GetCmcMap()
	if err != nil {
		return nil, err
	}
	var prices CoinPrices
	err = c.Get(&prices, "v1/cryptocurrency/listings/latest", url.Values{"limit": {"5000"}, "convert": {"USD"}})
	if err != nil {
		return
	}
	rates = normalizeRates(prices, cmap)
	return
}

func normalizeRates(prices CoinPrices, cmap cmcmap.CmcMapping) (rates blockatlas.Rates) {
	for _, price := range prices.Data {
		if price.Platform != nil {
			continue
		}
		currency := price.Symbol
		cmcCoin, _, err := cmap.GetCoin(price.Id)
		if err == nil {
			currency = cmcCoin.Symbol
		}
		rate := new(big.Float).Quo(big.NewFloat(1), big.NewFloat(price.Quote.USD.Price))
		rates = append(rates, blockatlas.Rate{
			Currency:  currency,
			Rate:      rate,
			Timestamp: price.LastUpdated.Unix(),
		})
	}
	return
}