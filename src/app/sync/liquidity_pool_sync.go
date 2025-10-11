package sync

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mumu/cryptoSwap/src/app/model"
	"github.com/mumu/cryptoSwap/src/core/chainclient/evm"
	"github.com/mumu/cryptoSwap/src/core/config"
	"github.com/mumu/cryptoSwap/src/core/ctx"
	"github.com/mumu/cryptoSwap/src/core/log"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// StartLiquidityPoolSync 启动流动性池事件监听
func StartLiquidityPoolSync(c context.Context) {
	var wg sync.WaitGroup
	for chainId := range ctx.Ctx.ChainMap {
		wg.Add(1)
		go func(chainId int) {
			defer wg.Done()
			evmClient := ctx.GetClient(chainId).(*evm.Evm)
			if evmClient == nil {
				log.Logger.Error("链客户端获取失败，无法启动流动性池监听", zap.Int("chain_id", chainId))
				return
			}

			log.Logger.Info("启动流动性池事件监听", zap.Int("chain_id", chainId))

			// 获取合约地址
			dexAddress := config.Conf.ContractCfg.DexAddress
			if dexAddress == "" {
				log.Logger.Error("DEX合约地址未配置", zap.Int("chain_id", chainId))
				return
			}

			// 查询数据库的最后一次区块
			var chain model.Chain
			err := ctx.Ctx.DB.Model(&model.Chain{}).Where("chain_id = ?", int64(chainId)).First(&chain).Error
			if err != nil {
				log.Logger.Error("查询链信息失败", zap.Int("chain_id", chainId), zap.Error(err))
				return
			}
			lastBlockNum := chain.LastBlockNum

			// 定义流动性池相关事件的topic hash
			// Swap事件: Swap(address indexed sender, uint amount0In, uint amount1In, uint amount0Out, uint amount1Out, address indexed to)
			swapTopic := crypto.Keccak256Hash([]byte("Swap(address,uint256,uint256,uint256,uint256,address)")).Hex()

			// Mint事件: Mint(address indexed sender, uint amount0, uint amount1)
			mintTopic := crypto.Keccak256Hash([]byte("Mint(address,uint256,uint256)")).Hex()

			// Burn事件: Burn(address indexed sender, uint amount0, uint amount1, address indexed to)
			burnTopic := crypto.Keccak256Hash([]byte("Burn(address,uint256,uint256,address)")).Hex()

			ticker := time.NewTicker(12 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-c.Done():
					log.Logger.Info("流动性池监听任务已停止", zap.Int("chain_id", chainId))
					return
				case <-ticker.C:
					// 获取当前块的高度
					currentBlock, err := evmClient.GetBlockNumber()
					if err != nil {
						log.Logger.Error("获取当前区块高度失败", zap.Int("chain_id", chainId), zap.Error(err))
						continue
					}

					targetBlockNum := currentBlock - 6
					if targetBlockNum <= lastBlockNum {
						log.Logger.Debug("当前区块高度不足，跳过本次执行",
							zap.Int("chain_id", chainId),
							zap.Uint64("last_block_num", lastBlockNum),
							zap.Uint64("current_block", currentBlock))
						continue
					}

					// 当断开链接很久时，分批次拉取日志，一次拉取1000个块的日志
					if targetBlockNum-lastBlockNum > 1000 {
						targetBlockNum = lastBlockNum + 1000
					}

					log.Logger.Info("开始监听流动性池事件",
						zap.Uint64("from_block", lastBlockNum),
						zap.Uint64("to_block", targetBlockNum))

					// 监听所有流动性池合约地址的事件
					contractAddresses := []string{dexAddress}
					topics := [][]common.Hash{
						{common.HexToHash(swapTopic), common.HexToHash(mintTopic), common.HexToHash(burnTopic)},
					}

					logs, err := evmClient.GetFilterLogsWithTopics(
						big.NewInt(int64(lastBlockNum)),
						big.NewInt(int64(targetBlockNum)),
						contractAddresses,
						topics)

					if err != nil {
						log.Logger.Error("获取流动性池事件日志失败", zap.Error(err))
						continue
					}

					if len(logs) == 0 {
						log.Logger.Debug("未获取到流动性池事件日志")
						// 更新区块号
						if err := updateLiquidityPoolBlockNumber(chainId, targetBlockNum); err != nil {
							log.Logger.Error("更新流动性池区块号失败", zap.Error(err))
						}
						continue
					}

					var liquidityPoolEvents []*model.LiquidityPoolEvent

					// 解析日志并保存到数据库
					for _, vLog := range logs {
						event := parseLiquidityPoolEvent(vLog, chainId)
						if event != nil {
							liquidityPoolEvents = append(liquidityPoolEvents, event)
						}
					}

					log.Logger.Info("解析流动性池事件成功", zap.Int("event_count", len(liquidityPoolEvents)))

					// 保存事件到数据库
					if err := saveLiquidityPoolEvents(liquidityPoolEvents, chainId, targetBlockNum); err != nil {
						log.Logger.Error("保存流动性池事件失败", zap.Error(err))
					} else {
						lastBlockNum = targetBlockNum + 1
					}
				}
			}
		}(chainId)
	}
	// 一直等待
	wg.Wait()
}

// parseLiquidityPoolEvent 解析流动性池事件
func parseLiquidityPoolEvent(vLog types.Log, chainId int) *model.LiquidityPoolEvent {
	if len(vLog.Topics) == 0 {
		return nil
	}

	topic0 := vLog.Topics[0].Hex()

	// Swap事件解析
	if topic0 == crypto.Keccak256Hash([]byte("Swap(address,uint256,uint256,uint256,uint256,address)")).Hex() {
		return parseSwapEvent(vLog, chainId)
	}

	// Mint事件解析
	if topic0 == crypto.Keccak256Hash([]byte("Mint(address,uint256,uint256)")).Hex() {
		return parseMintEvent(vLog, chainId)
	}

	// Burn事件解析
	if topic0 == crypto.Keccak256Hash([]byte("Burn(address,uint256,uint256,address)")).Hex() {
		return parseBurnEvent(vLog, chainId)
	}

	return nil
}

// parseSwapEvent 解析Swap事件
func parseSwapEvent(vLog types.Log, chainId int) *model.LiquidityPoolEvent {
	if len(vLog.Topics) < 3 || len(vLog.Data) < 128 {
		return nil
	}

	// Swap(address indexed sender, uint amount0In, uint amount1In, uint amount0Out, uint amount1Out, address indexed to)
	sender := common.BytesToAddress(vLog.Topics[1].Bytes()).Hex()
	_ = common.BytesToAddress(vLog.Topics[2].Bytes()).Hex() // to address (not used in current implementation)

	data := vLog.Data
	amount0In := new(big.Int).SetBytes(common.TrimLeftZeroes(data[0:32]))
	amount1In := new(big.Int).SetBytes(common.TrimLeftZeroes(data[32:64]))
	amount0Out := new(big.Int).SetBytes(common.TrimLeftZeroes(data[64:96]))
	amount1Out := new(big.Int).SetBytes(common.TrimLeftZeroes(data[96:128]))

	return &model.LiquidityPoolEvent{
		ChainId:     int64(chainId),
		TxHash:      vLog.TxHash.Hex(),
		BlockNumber: int64(vLog.BlockNumber),
		EventType:   "Swap",
		PoolAddress: vLog.Address.Hex(),
		UserAddress: sender,
		Amount0In:   amount0In.String(),
		Amount1In:   amount1In.String(),
		Amount0Out:  amount0Out.String(),
		Amount1Out:  amount1Out.String(),
	}
}

// parseMintEvent 解析Mint事件
func parseMintEvent(vLog types.Log, chainId int) *model.LiquidityPoolEvent {
	if len(vLog.Topics) < 2 || len(vLog.Data) < 64 {
		return nil
	}

	// Mint(address indexed sender, uint amount0, uint amount1)
	sender := common.BytesToAddress(vLog.Topics[1].Bytes()).Hex()

	data := vLog.Data
	amount0 := new(big.Int).SetBytes(common.TrimLeftZeroes(data[0:32]))
	amount1 := new(big.Int).SetBytes(common.TrimLeftZeroes(data[32:64]))

	return &model.LiquidityPoolEvent{
		ChainId:     int64(chainId),
		TxHash:      vLog.TxHash.Hex(),
		BlockNumber: int64(vLog.BlockNumber),
		EventType:   "AddLiquidity",
		PoolAddress: vLog.Address.Hex(),
		UserAddress: sender,
		Amount0In:   amount0.String(),
		Amount1In:   amount1.String(),
		Amount0Out:  "0",
		Amount1Out:  "0",
	}
}

// parseBurnEvent 解析Burn事件
func parseBurnEvent(vLog types.Log, chainId int) *model.LiquidityPoolEvent {
	if len(vLog.Topics) < 3 || len(vLog.Data) < 64 {
		return nil
	}

	// Burn(address indexed sender, uint amount0, uint amount1, address indexed to)
	sender := common.BytesToAddress(vLog.Topics[1].Bytes()).Hex()
	_ = common.BytesToAddress(vLog.Topics[2].Bytes()).Hex() // to address (not used in current implementation)

	data := vLog.Data
	amount0 := new(big.Int).SetBytes(common.TrimLeftZeroes(data[0:32]))
	amount1 := new(big.Int).SetBytes(common.TrimLeftZeroes(data[32:64]))

	return &model.LiquidityPoolEvent{
		ChainId:     int64(chainId),
		TxHash:      vLog.TxHash.Hex(),
		BlockNumber: int64(vLog.BlockNumber),
		EventType:   "RemoveLiquidity",
		PoolAddress: vLog.Address.Hex(),
		UserAddress: sender,
		Amount0In:   "0",
		Amount1In:   "0",
		Amount0Out:  amount0.String(),
		Amount1Out:  amount1.String(),
	}
}

// saveLiquidityPoolEvents 保存流动性池事件到数据库
func saveLiquidityPoolEvents(events []*model.LiquidityPoolEvent, chainId int, targetBlockNum uint64) error {
	if len(events) == 0 {
		return updateLiquidityPoolBlockNumber(chainId, targetBlockNum)
	}

	return ctx.Ctx.DB.Transaction(func(tx *gorm.DB) error {
		// 批量插入流动性池事件
		if err := tx.CreateInBatches(events, 100).Error; err != nil {
			log.Logger.Error("批量插入流动性池事件失败", zap.Error(err))
			return err
		}

		// 更新流动性池信息
		if err := updateLiquidityPoolInfo(tx, events); err != nil {
			log.Logger.Error("更新流动性池信息失败", zap.Error(err))
			return err
		}

		// 更新区块号
		return updateLiquidityPoolBlockNumber(chainId, targetBlockNum)
	})
}

// updateLiquidityPoolInfo 更新流动性池信息
func updateLiquidityPoolInfo(tx *gorm.DB, events []*model.LiquidityPoolEvent) error {
	// 按池子地址分组
	poolEvents := make(map[string][]*model.LiquidityPoolEvent)
	for _, event := range events {
		poolEvents[event.PoolAddress] = append(poolEvents[event.PoolAddress], event)
	}

	for poolAddress, poolEventList := range poolEvents {
		// 检查池子是否存在
		var pool model.LiquidityPool
		err := tx.Where("pool_address = ? AND chain_id = ?", poolAddress, poolEventList[0].ChainId).First(&pool).Error

		if err == gorm.ErrRecordNotFound {
			// 创建新的流动性池记录
			pool = model.LiquidityPool{
				ChainId:      poolEventList[0].ChainId,
				PoolAddress:  poolAddress,
				LastBlockNum: poolEventList[len(poolEventList)-1].BlockNumber,
				IsActive:     true,
			}

			if err := tx.Create(&pool).Error; err != nil {
				log.Logger.Error("创建流动性池记录失败", zap.Error(err))
				return err
			}
		} else if err != nil {
			log.Logger.Error("查询流动性池记录失败", zap.Error(err))
			return err
		} else {
			// 更新现有池子的区块号
			if err := tx.Model(&pool).Update("last_block_num", poolEventList[len(poolEventList)-1].BlockNumber).Error; err != nil {
				log.Logger.Error("更新流动性池区块号失败", zap.Error(err))
				return err
			}
		}

		// 更新交易计数
		if err := tx.Model(&pool).Update("tx_count", gorm.Expr("tx_count + ?", len(poolEventList))).Error; err != nil {
			log.Logger.Error("更新流动性池交易计数失败", zap.Error(err))
			return err
		}
	}

	return nil
}

// updateLiquidityPoolBlockNumber 更新流动性池监听的区块号
func updateLiquidityPoolBlockNumber(chainId int, blockNumber uint64) error {
	// 这里可以创建一个专门的表来记录流动性池监听的区块号
	// 或者复用现有的chain表，添加一个字段来记录流动性池监听的区块号
	return ctx.Ctx.DB.Model(&model.Chain{}).Where("chain_id = ?", int64(chainId)).Update("last_block_num", blockNumber).Error
}
