package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/binance-chain/bsc-relayer/executor/relayerincentivize"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	startHeight = 4696000 //4696000
	endHeight   = 4698000 //5124228
)

type Reward struct {
	Relayer common.Address
	Amount  *big.Int
}

func main() {
	var crossChainABI, _ = abi.JSON(strings.NewReader(relayerincentivize.RelayerincentivizeABI))
	txFeeMap := make(map[common.Address]*big.Int)
	relayerAccRewardMap := make(map[common.Address]*big.Int)
	bscClient, err := ethclient.Dial("wss://bsc-ws-node.nariox.org:443")
	if err != nil {
		panic(fmt.Sprintf("new eth client error: %s", err.Error()))
	}

	for height := int64(startHeight); height < int64(endHeight); height++ {

		ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		header, err := bscClient.HeaderByNumber(ctxWithTimeout, big.NewInt(height))
		if err != nil {
			panic(err.Error())
		}

		block, err := bscClient.BlockByHash(ctxWithTimeout, header.Hash())
		if err != nil {
			panic(err.Error())
		}

		if len(block.Body().Transactions) > 1 {
			for idx := 0; idx < len(block.Body().Transactions)-1; idx++ {
				tx := block.Body().Transactions[idx]

				if tx == nil || tx.To() == nil {
					continue
				}
				eip155Signer := types.NewEIP155Signer(big.NewInt(56))
				sender, err := eip155Signer.Sender(tx)
				if err != nil {
					panic(err.Error())
				}

				if tx.To().String() == "0x0000000000000000000000000000000000002000" {
					txRecipient, err := bscClient.TransactionReceipt(ctxWithTimeout, tx.Hash())
					if err != nil {
						fmt.Println("Get transaction recipient failure")
						continue
					}
					txFee := big.NewInt(0).Mul(tx.GasPrice(), big.NewInt(int64(txRecipient.GasUsed)))
					accFee, ok := txFeeMap[sender]
					if !ok {
						accFee = big.NewInt(0)
					}
					accFee = big.NewInt(0).Add(accFee, txFee)
					txFeeMap[sender] = accFee
					fmt.Printf("height %d, relay package tx: %s, Relayer: %s, tx status: %d, txFee: %s, accTxFee %s\n", height, tx.Hash().String(), sender.String(),
						txRecipient.Status, txFee.String(), accFee.String())

				} else if tx.To().String() == "0x0000000000000000000000000000000000001003" {
					txRecipient, err := bscClient.TransactionReceipt(ctxWithTimeout, tx.Hash())
					if err != nil {
						fmt.Println("Get transaction recipient failure")
						continue
					}
					txFee := big.NewInt(0).Mul(tx.GasPrice(), big.NewInt(int64(txRecipient.GasUsed)))
					accFee, ok := txFeeMap[sender]
					if !ok {
						accFee = big.NewInt(0)
					}
					accFee = big.NewInt(0).Add(accFee, txFee)
					txFeeMap[sender] = accFee
					fmt.Printf("height %d, relay header tx: %s, Relayer: %s, tx status: %d, txFee: %s, accTxFee %s\n", height, tx.Hash().String(), sender.String(),
						txRecipient.Status, txFee.String(), accFee.String())
				} else if tx.To().String() == "0x0000000000000000000000000000000000001005" {
					txRecipient, err := bscClient.TransactionReceipt(ctxWithTimeout, tx.Hash())
					if err != nil {
						fmt.Println("Get transaction recipient failure")
						continue
					}
					if txRecipient.Status == 1 {
						var reward Reward
						err = crossChainABI.Unpack(&reward, "rewardToRelayer", txRecipient.Logs[0].Data)
						if err != nil {
							fmt.Printf("Decode rewardToRelayer event failure, err: %s, data: %s\n", err.Error(), hex.EncodeToString(txRecipient.Logs[0].Data))
							continue
						}
						relayerAccReward, ok := relayerAccRewardMap[sender]
						if !ok {
							relayerAccReward = big.NewInt(0)
						}
						relayerAccReward = big.NewInt(0).Add(relayerAccReward, reward.Amount)
						relayerAccRewardMap[sender] = relayerAccReward
						fmt.Printf("Reward to %s, Amount: %s, Accumulated reward Amount: %s\n", reward.Relayer.String(), reward.Amount.String(), relayerAccReward.String())
					}
				}
			}
		}
	}
	for relayer, fee := range txFeeMap {
		fmt.Printf("Relayer %s, acc fee %s\n", relayer.String(), fee.String())
	}
	for relayer, reward := range relayerAccRewardMap {
		fmt.Printf("Relayer %s, acc reward %s\n", relayer.String(), reward.String())
	}
}
