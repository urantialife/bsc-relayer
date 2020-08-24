package relayer

import (
	"time"

	"github.com/binance-chain/bsc-relayer/common"
	"github.com/binance-chain/bsc-relayer/executor"
	cmn "github.com/tendermint/tendermint/libs/common"
)

func RelayerDaemon(bbcExecutor *executor.BBCExecutor, bscExecutor *executor.BSCExecutor, startHeight uint64, curValidatorsHash cmn.HexBytes) {
	var tashSet *common.TaskSet
	var err error
	height := startHeight
	common.Logger.Info("Start relayer daemon")
	for {
		common.Logger.Infof("step 1 time %vvi 	", time.Now())
		tashSet, curValidatorsHash, err = bbcExecutor.MonitorCrossChainPackage(int64(height), curValidatorsHash)
		common.Logger.Infof("step 2 time %vvi 	", time.Now())
		if err != nil {
			sleepTime := time.Duration(bbcExecutor.Config.BBCConfig.SleepMillisecondForWaitBlock * int64(time.Millisecond))
			common.Logger.Infof("step 3 time %vvi 	", time.Now())
			time.Sleep(sleepTime)
			continue
		}

		// Found validator set change
		if len(tashSet.TaskList) > 0 {
			if tashSet.TaskList[0].ChannelID == executor.PureHeaderSyncChannelID {
				txHash, err := bscExecutor.SyncTendermintLightClientHeader(tashSet.Height)
				common.Logger.Infof("step 4 time %vvi 	", time.Now())

				if err != nil {
					common.Logger.Error(err.Error())
				}
				common.Logger.Infof("Syncing header for validatorset update on Binance Chain, height:%d, txHash: %s", tashSet.Height, txHash.String())
			}
		}

		if height%bbcExecutor.Config.BBCConfig.BlockIntervalForCleanUpUndeliveredPackages == 0 {
			err := CleanPreviousPackages(bbcExecutor, bscExecutor, height)
			common.Logger.Infof("step 5 time %vvi 	", time.Now())

			if err != nil {
				common.Logger.Error(err.Error())
			}
			height++
			continue // packages have been delivered on cleanup
		}

		if len(tashSet.TaskList) == 0 || (len(tashSet.TaskList) == 1 && tashSet.TaskList[0].ChannelID == executor.PureHeaderSyncChannelID) {
			height++
			continue // skip this height
		}

		txHash, err := bscExecutor.SyncTendermintLightClientHeader(tashSet.Height + 1)
		common.Logger.Infof("step 6 time %vvi 	", time.Now())

		if err != nil {
			common.Logger.Error(err.Error())
			continue // try again for this height
		}
		common.Logger.Infof("Syncing header: %d, txHash: %s", tashSet.Height+1, txHash.String())

		for _, task := range tashSet.TaskList {
			_, err := bscExecutor.RelayCrossChainPackage(task.ChannelID, task.Sequence, tashSet.Height)
			if err != nil {
				common.Logger.Error(err.Error())
				continue
			}
		}
		height++
	}
}
