package main

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	height := int64(5037643)
	bscClient, err := ethclient.Dial("https://bsc-dataseed1.binance.org:443")
	if err != nil {
		panic("new eth client error")
	}

	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	header, err := bscClient.HeaderByNumber(ctxWithTimeout, big.NewInt(height))
	if err != nil {
		panic(err.Error())
	}

	txCount, err := bscClient.TransactionCount(ctxWithTimeout, header.Hash())
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("txCount %d\n", txCount)

	if txCount > 1 {
		for idx := uint(0); idx < txCount-1; idx++ {
			tx, err := bscClient.TransactionInBlock(ctxWithTimeout, header.Hash(), idx)
			if err != nil {
				panic(err.Error())
			}
			bscClient.SendTransaction(ctxWithTimeout, tx)
			fmt.Printf("Idx %d, txHash %s\n", idx, tx.Hash().String())
		}
	}

}
