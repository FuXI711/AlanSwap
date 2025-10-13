package sync

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mumu/cryptoSwap/src/app/model"
	"github.com/mumu/cryptoSwap/src/core/ctx"
	"github.com/mumu/cryptoSwap/src/core/log"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

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

// getPoolTokenAddresses 获取池子的代币地址
func getPoolTokenAddresses(poolAddress string, chainId int) (string, string) {
	var pool model.LiquidityPool
	err := ctx.Ctx.DB.Where("pool_address = ? AND chain_id = ?", poolAddress, chainId).First(&pool).Error
	if err != nil {
		log.Logger.Warn("查询流动性池代币地址失败",
			zap.String("pool_address", poolAddress),
			zap.Int("chain_id", chainId),
			zap.Error(err))
		// 返回默认值，避免空字符串
		return "0x0000000000000000000000000000000000000000", "0x0000000000000000000000000000000000000000"
	}
	return pool.Token0Address, pool.Token1Address
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
	// 获取池子的代币地址
	token0Address, token1Address := getPoolTokenAddresses(vLog.Address.Hex(), chainId)
	return &model.LiquidityPoolEvent{
		ChainId:       int64(chainId),
		TxHash:        vLog.TxHash.Hex(),
		BlockNumber:   int64(vLog.BlockNumber),
		EventType:     "Swap",
		PoolAddress:   vLog.Address.Hex(),
		Token0Address: token0Address,
		Token1Address: token1Address,
		UserAddress:   sender,
		Amount0In:     amount0In.String(),  // 改为字符串
		Amount1In:     amount1In.String(),  // 改为字符串
		Amount0Out:    amount0Out.String(), // 改为字符串
		Amount1Out:    amount1Out.String(), // 改为字符串
		Reserve0:      "0",                 // 改为字符串
		Reserve1:      "0",                 // 改为字符串
		Price:         "0",                 // 改为字符串
		Liquidity:     "0",                 // 改为字符串
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
	// 获取池子的代币地址
	token0Address, token1Address := getPoolTokenAddresses(vLog.Address.Hex(), chainId)
	return &model.LiquidityPoolEvent{
		ChainId:       int64(chainId),
		TxHash:        vLog.TxHash.Hex(),
		BlockNumber:   int64(vLog.BlockNumber),
		EventType:     "AddLiquidity",
		PoolAddress:   vLog.Address.Hex(),
		Token0Address: token0Address,
		Token1Address: token1Address,
		UserAddress:   sender,
		Amount0In:     amount0.String(), // 改为字符串
		Amount1In:     amount1.String(), // 改为字符串
		Amount0Out:    "0",              // 改为字符串
		Amount1Out:    "0",              // 改为字符串
		Reserve0:      "0",              // 改为字符串
		Reserve1:      "0",              // 改为字符串
		Price:         "0",              // 改为字符串
		Liquidity:     "0",              // 改为字符串
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
	// 获取池子的代币地址
	token0Address, token1Address := getPoolTokenAddresses(vLog.Address.Hex(), chainId)
	return &model.LiquidityPoolEvent{
		ChainId:       int64(chainId),
		TxHash:        vLog.TxHash.Hex(),
		BlockNumber:   int64(vLog.BlockNumber),
		EventType:     "RemoveLiquidity",
		PoolAddress:   vLog.Address.Hex(),
		Token0Address: token0Address,
		Token1Address: token1Address,
		UserAddress:   sender,
		Amount0In:     "0",              // 改为字符串
		Amount1In:     "0",              // 改为字符串
		Amount0Out:    amount0.String(), // 改为字符串
		Amount1Out:    amount1.String(), // 改为字符串
		Reserve0:      "0",              // 改为字符串
		Reserve1:      "0",              // 改为字符串
		Price:         "0",              // 改为字符串
		Liquidity:     "0",              // 改为字符串
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
				ChainId:        poolEventList[0].ChainId,
				PoolAddress:    poolAddress,
				Token0Address:  "0x0000000000000000000000000000000000000000", // 暂时留空，需要从合约获取
				Token1Address:  "0x0000000000000000000000000000000000000000", // 暂时留空，需要从合约获取
				Token0Symbol:   "",                                           // 暂时留空
				Token1Symbol:   "",                                           // 暂时留空
				Token0Decimals: 0,                                            // 默认值
				Token1Decimals: 0,                                            // 默认值
				Reserve0:       "0",                                          // 默认值
				Reserve1:       "0",                                          // 默认值
				TotalSupply:    "0",                                          // 默认值
				Price:          "0",                                          // 默认值
				Volume24h:      "0",                                          // 默认值
				TxCount:        0,                                            // 默认值
				LastBlockNum:   poolEventList[len(poolEventList)-1].BlockNumber,
				IsActive:       true,
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
	// 更新流动性池服务配置的区块号
	return ctx.Ctx.DB.Model(&model.Chain{}).Where("chain_id = ? ", int64(chainId)).Update("last_block_num", blockNumber).Error
}
