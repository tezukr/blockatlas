package binance

import (
	log "github.com/sirupsen/logrus"
	"github.com/trustwallet/blockatlas/pkg/blockatlas"
	"github.com/trustwallet/blockatlas/pkg/logger"
	"github.com/trustwallet/blockatlas/pkg/numbers"
	"strings"
	"sync"

	"github.com/trustwallet/blockatlas/coin"
)

func (p *Platform) GetTxsByAddress(address string) (blockatlas.TxPage, error) {
	return p.GetTokenTxsByAddress(address, "")
}

func (p *Platform) GetTokenTxsByAddress(address, token string) (blockatlas.TxPage, error) {
	var transferTypes = []TxType{TxTransfer}
	var wg sync.WaitGroup
	out := make(chan []Tx, len(transferTypes))
	wg.Add(len(transferTypes))
	for _, t := range transferTypes {
		go func(txType TxType, address, token string, wg *sync.WaitGroup) {
			defer wg.Done()
			txs, err := p.client.GetAddressAssetTransactions(address, token, string(txType))
			if err != nil {
				log.Error("GetAddressAssetTransactions : ", err, logger.Params{"txType": txType, "address": address, "token": token})
				return
			}
			out <- txs
		}(t, address, token, &wg)
	}
	wg.Wait()
	close(out)

	srcTx := make([]Tx, 0)
	for r := range out {
		srcTx = append(srcTx, r...)
	}

	return NormalizeTxs(srcTx, address), nil
}

// Converts multiple transactions
func NormalizeTxs(srcTxs []Tx, address string) (txs []blockatlas.Tx) {
	for _, srcTx := range srcTxs {
		tx, ok := NormalizeTx(srcTx, address)
		if !ok {
			continue
		}
		txs = append(txs, tx...)
	}
	return
}

// Converts a Binance transaction into the generic model
func NormalizeTx(t Tx, address string) (blockatlas.TxPage, bool) {
	rawTx := blockatlas.Tx{
		ID:       t.TxHash,
		Coin:     coin.BNB,
		From:     t.FromAddr,
		To:       t.ToAddr,
		Fee:      blockatlas.Amount(t.getFee()),
		Date:     t.blockTimestamp(),
		Block:    t.BlockHeight,
		Status:   t.getStatus(),
		Error:    t.getError(),
		Sequence: t.Sequence,
		Memo:     t.Memo,
	}
	rawTx.Direction = rawTx.GetTransactionDirection(address)

	switch t.Type {
	case TxTransfer:
		normalized, ok := normalizeTransfer(rawTx, t)
		if !ok {
			return blockatlas.TxPage{}, false
		}
		return blockatlas.TxPage{normalized}, true
	}

	return blockatlas.TxPage{}, false
}

func normalizeTransfer(t blockatlas.Tx, srcTx Tx) (blockatlas.Tx, bool) {
	bnbCoin := coin.Coins[coin.BNB]
	value := blockatlas.Amount(numbers.DecimalExp(srcTx.Value, 8))

	// Condition for native transfer e.g.: BNB
	if srcTx.Asset == bnbCoin.Symbol {
		t.Type = blockatlas.TxTransfer
		t.Meta = blockatlas.Transfer{
			Decimals: bnbCoin.Decimals,
			Symbol:   bnbCoin.Symbol,
			Value:    value,
		}
		return t, true
	}

	// Condition for BEP2 token transfer e.g.: TWT-8C2
	t.Type = blockatlas.TxNativeTokenTransfer
	t.Meta = blockatlas.NativeTokenTransfer{
		Decimals: bnbCoin.Decimals,
		From:     srcTx.FromAddr,
		Symbol:   TokenSymbol(srcTx.Asset),
		To:       srcTx.ToAddr,
		TokenID:  srcTx.Asset,
		Value:    value,
		Name:     "",
	}
	return t, true
}

// Extract BEP2 token symbol from asset name e.g: TWT-8C2 => TWT
func TokenSymbol(asset string) string {
	s := strings.Split(asset, "-")
	if len(s) > 1 {
		return s[0]
	}
	return asset
}
