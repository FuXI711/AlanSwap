package api

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/mumu/cryptoSwap/src/abi" // 添加abi包导入
	"github.com/mumu/cryptoSwap/src/app/model"
	"github.com/mumu/cryptoSwap/src/app/service"
	"github.com/mumu/cryptoSwap/src/core/ctx"
	"github.com/mumu/cryptoSwap/src/core/log"
	"github.com/mumu/cryptoSwap/src/core/result"
	"go.uber.org/zap"
)

type LiquidityPoolApi struct {
	svc *service.LiquidityPoolService
}

func NewLiquidityPoolApi() *LiquidityPoolApi {
	return &LiquidityPoolApi{
		svc: service.NewLiquidityPoolService(),
	}
}

// GetRPCURL 根据chainId获取RPC端点
func GetRPCURL(chainId int) string {
	rpcMap := map[int]string{
		11155111: "https://sepolia.infura.io/v3/a6dc9c34bc0c480e9acf43659fc37b1b", // Ropsten Testnet
		1:        "https://mainnet.infura.io/v3/YOUR_PROJECT_ID",                  // Ethereum Mainnet
		5:        "https://goerli.infura.io/v3/YOUR_PROJECT_ID",                   // Goerli Testnet
		56:       "https://bsc-dataseed.binance.org/",                             // BSC Mainnet (PancakeSwap)
		97:       "https://data-seed-prebsc-1-s1.binance.org:8545/",               // BSC Testnet
		137:      "https://polygon-rpc.com/",                                      // Polygon Mainnet
		80001:    "https://rpc-mumbai.matic.today",                                // Polygon Testnet
		42161:    "https://arb1.arbitrum.io/rpc",                                  // Arbitrum Mainnet
		421613:   "https://goerli-rollup.arbitrum.io/rpc",                         // Arbitrum Testnet
		10:       "https://mainnet.optimism.io",                                   // Optimism Mainnet
		420:      "https://goerli.optimism.io",                                    // Optimism Testnet
	}
	return rpcMap[chainId]
}

// GetPoolReserves 直接从 Uniswap V2 池子合约获取储备量
func GetPoolReserves(poolAddress string, chainId int) (reserve0, reserve1, totalSupply *big.Int, err error) {
	rpcURL := GetRPCURL(chainId)
	if rpcURL == "" {
		return nil, nil, nil, fmt.Errorf("不支持的 chainId: %d", chainId)
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, nil, nil, err
	}
	defer client.Close()

	contractAddress := common.HexToAddress(poolAddress)

	// 使用ABI管理器单例获取UniswapV2Pair ABI
	abiManager := abi.GetABIManager()
	uniswapV2PairABI, exists := abiManager.GetABI("UniswapV2Pair")
	if !exists {
		return nil, nil, nil, fmt.Errorf("UniswapV2Pair ABI 未找到")
	}

	// 调用 getReserves 函数
	reservesData, err := uniswapV2PairABI.Pack("getReserves")
	if err != nil {
		return nil, nil, nil, err
	}

	msg := ethereum.CallMsg{
		To:   &contractAddress,
		Data: reservesData,
	}

	reservesResult, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	var reserves struct {
		Reserve0           *big.Int
		Reserve1           *big.Int
		BlockTimestampLast uint32
	}

	err = uniswapV2PairABI.UnpackIntoInterface(&reserves, "getReserves", reservesResult)
	if err != nil {
		return nil, nil, nil, err
	}

	// 调用 totalSupply 函数
	supplyData, err := uniswapV2PairABI.Pack("totalSupply")
	if err != nil {
		return nil, nil, nil, err
	}

	msg.Data = supplyData
	supplyResult, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	err = uniswapV2PairABI.UnpackIntoInterface(&totalSupply, "totalSupply", supplyResult)
	if err != nil {
		return nil, nil, nil, err
	}

	return reserves.Reserve0, reserves.Reserve1, totalSupply, nil
}

// GetUserLPTokenBalance 获取用户在 Uniswap V2 池子中的 LP 代币余额
func GetUserLPTokenBalance(poolAddress, userAddress string, chainId int) (*big.Int, error) {
	rpcURL := GetRPCURL(chainId)
	if rpcURL == "" {
		return nil, fmt.Errorf("不支持的 chainId: %d", chainId)
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	contractAddress := common.HexToAddress(poolAddress)
	ownerAddress := common.HexToAddress(userAddress)

	// 使用ABI管理器单例获取UniswapV2Pair ABI
	abiManager := abi.GetABIManager()
	uniswapV2PairABI, exists := abiManager.GetABI("UniswapV2Pair")
	if !exists {
		return nil, fmt.Errorf("UniswapV2Pair ABI 未找到")
	}

	// 调用 balanceOf 函数
	balanceData, err := uniswapV2PairABI.Pack("balanceOf", ownerAddress)

	msg := ethereum.CallMsg{
		To:   &contractAddress,
		Data: balanceData,
	}

	balanceResult, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return nil, err
	}

	var balance *big.Int
	err = uniswapV2PairABI.UnpackIntoInterface(&balance, "balanceOf", balanceResult)
	if err != nil {
		return nil, err
	}

	return balance, nil
}

// GetUserLiquidityPoolsByLPToken 根据用户LP代币余额获取参与的流动性池
func GetUserLiquidityPoolsByLPToken(c *gin.Context) {
	userAddress := c.Query("userAddress")
	chainIdStr := c.Query("chainId")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("pageSize", "20")

	if userAddress == "" {
		result.Error(c, result.InvalidParameter)
		return
	}

	chainId, err := strconv.ParseInt(chainIdStr, 10, 64)
	if err != nil && chainIdStr != "" {
		result.Error(c, result.InvalidParameter)
		return
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// 1. 查询所有流动性池信息
	var allPools []model.LiquidityPool
	query := ctx.Ctx.DB.Model(&model.LiquidityPool{}).Where("is_active = ?", true)
	if chainId > 0 {
		query = query.Where("chain_id = ?", chainId)
	}

	if err := query.Find(&allPools).Error; err != nil {
		result.Error(c, result.DBQueryFailed)
		return
	}

	if len(allPools) == 0 {
		result.OK(c, gin.H{
			"pools":    []model.LiquidityPool{},
			"total":    0,
			"page":     page,
			"pageSize": pageSize,
		})
		return
	}

	// 2. 查询每个池子的LP代币余额
	var userPools []model.LiquidityPool
	for _, pool := range allPools {
		// 直接调用Uniswap V2池子合约的balanceOf函数
		balance, err := GetUserLPTokenBalance(pool.PoolAddress, userAddress, int(chainId))
		if err != nil {
			// 查询失败，跳过这个池子
			log.Logger.Warn("查询LP代币余额失败",
				zap.String("pool", pool.PoolAddress),
				zap.Error(err))
			continue
		}

		// 如果余额大于0，添加到结果中
		if balance.Cmp(big.NewInt(0)) > 0 {
			userPools = append(userPools, pool)
		}
	}

	// 3. 分页处理
	total := len(userPools)
	start := offset
	end := offset + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	var pagedPools []model.LiquidityPool
	if start < end {
		pagedPools = userPools[start:end]
	}

	result.OK(c, gin.H{
		"pools":    pagedPools,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetUserLiquidityPools 根据用户地址获取参与的流动性池
func GetUserLiquidityPools(c *gin.Context) {
	userAddress := c.Query("userAddress")
	chainIdStr := c.Query("chainId")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("pageSize", "20")

	if userAddress == "" {
		result.Error(c, result.InvalidParameter)
		return
	}

	chainId, err := strconv.ParseInt(chainIdStr, 10, 64)
	if err != nil && chainIdStr != "" {
		result.Error(c, result.InvalidParameter)
		return
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// 查询用户参与过的流动性池地址（去重）
	var poolAddresses []string
	subQuery := ctx.Ctx.DB.Model(&model.LiquidityPoolEvent{}).
		Select("DISTINCT pool_address").
		Where("user_address = ?", userAddress)

	if chainId > 0 {
		subQuery = subQuery.Where("chain_id = ?", chainId)
	}

	if err := subQuery.Pluck("pool_address", &poolAddresses).Error; err != nil {
		result.Error(c, result.DBQueryFailed)
		return
	}

	if len(poolAddresses) == 0 {
		result.OK(c, gin.H{
			"pools":    []model.LiquidityPool{},
			"total":    0,
			"page":     page,
			"pageSize": pageSize,
		})
		return
	}

	// 根据池子地址查询完整的流动性池信息
	var pools []model.LiquidityPool
	var total int64

	query := ctx.Ctx.DB.Model(&model.LiquidityPool{}).
		Where("pool_address IN (?) AND is_active = ?", poolAddresses, true)

	if chainId > 0 {
		query = query.Where("chain_id = ?", chainId)
	}

	if err := query.Count(&total).Error; err != nil {
		result.Error(c, result.DBQueryFailed)
		return
	}

	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&pools).Error; err != nil {
		result.Error(c, result.DBQueryFailed)
		return
	}

	result.OK(c, gin.H{
		"pools":    pools,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetLiquidityPools 获取流动性池列表
func GetLiquidityPools(c *gin.Context) {
	chainIdStr := c.Query("chainId")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("pageSize", "20")

	chainId, err := strconv.ParseInt(chainIdStr, 10, 64)
	if err != nil {
		result.Error(c, result.InvalidParameter)
		return
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	var pools []model.LiquidityPool
	var total int64

	query := ctx.Ctx.DB.Model(&model.LiquidityPool{}).Where("is_active = ?", true)
	if chainId > 0 {
		query = query.Where("chain_id = ?", chainId)
	}

	if err := query.Count(&total).Error; err != nil {
		result.Error(c, result.DBQueryFailed)
		return
	}

	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&pools).Error; err != nil {
		result.Error(c, result.DBQueryFailed)
		return
	}

	result.OK(c, gin.H{
		"pools":    pools,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetLiquidityPoolEvents 获取流动性池事件列表
func (lp *LiquidityPoolApi) GetLiquidityPoolEvents(c *gin.Context) {
	poolAddress := c.Query("poolAddress")
	eventType := c.Query("eventType")
	userAddress := c.Query("userAddress")
	chainIdStr := c.Query("chainId")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("pageSize", "20")

	chainId, err := strconv.ParseInt(chainIdStr, 10, 64)
	if err != nil && chainIdStr != "" {
		result.Error(c, result.InvalidParameter)
		return
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	var events []model.LiquidityPoolEvent
	var total int64

	query := ctx.Ctx.DB.Model(&model.LiquidityPoolEvent{})

	if chainId > 0 {
		query = query.Where("chain_id = ?", chainId)
	}
	if poolAddress != "" {
		query = query.Where("pool_address = ?", poolAddress)
	}
	if eventType != "" {
		query = query.Where("event_type = ?", eventType)
	}
	if userAddress != "" {
		query = query.Where("user_address = ?", userAddress)
	}

	if err := query.Count(&total).Error; err != nil {
		result.Error(c, result.DBQueryFailed)
		return
	}

	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&events).Error; err != nil {
		result.Error(c, result.DBQueryFailed)
		return
	}

	result.OK(c, gin.H{
		"events":   events,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetLiquidityPoolStats 获取流动性池统计信息
func (lp *LiquidityPoolApi) GetLiquidityPoolStats(c *gin.Context) {
	chainIdStr := c.Query("chainId")

	var chainId int64
	if chainIdStr != "" {
		var err error
		chainId, err = strconv.ParseInt(chainIdStr, 10, 64)
		if err != nil {
			result.Error(c, result.InvalidParameter)
			return
		}
	}

	query := ctx.Ctx.DB.Model(&model.LiquidityPool{}).Where("is_active = ?", true)
	if chainId > 0 {
		query = query.Where("chain_id = ?", chainId)
	}

	var totalPools int64
	if err := query.Count(&totalPools).Error; err != nil {
		result.Error(c, result.DBQueryFailed)
		return
	}

	// 查询事件统计
	eventQuery := ctx.Ctx.DB.Model(&model.LiquidityPoolEvent{})
	if chainId > 0 {
		eventQuery = eventQuery.Where("chain_id = ?", chainId)
	}

	var totalEvents int64
	if err := eventQuery.Count(&totalEvents).Error; err != nil {
		result.Error(c, result.DBQueryFailed)
		return
	}

	// 查询今日事件数
	var todayEvents int64
	if err := eventQuery.Where("DATE(created_at) = CURDATE()").Count(&todayEvents).Error; err != nil {
		result.Error(c, result.DBQueryFailed)
		return
	}

	result.OK(c, gin.H{
		"poolPair":  totalPools,
		"24hVolume": totalEvents,
	})
}

// GetPoolPerformance 返回用户相关池子的 24 小时交易量表现
func (lp *LiquidityPoolApi) GetPoolPerformance(c *gin.Context) {
	walletAddress := c.Query("walletAddress")
	if walletAddress == "" {
		result.Error(c, result.InvalidParameter)
		return
	}

	// 查询用户参与过的流动性池地址（按事件记录去重）
	var poolAddresses []string
	subQuery := ctx.Ctx.DB.Model(&model.LiquidityPoolEvent{}).
		Select("DISTINCT pool_address").
		Where("user_address = ?", walletAddress)
	if err := subQuery.Pluck("pool_address", &poolAddresses).Error; err != nil {
		result.Error(c, result.DBQueryFailed)
		return
	}

	// 如果没有记录，直接返回空列表
	if len(poolAddresses) == 0 {
		result.OK(c, gin.H{
			"poolPerformance": []gin.H{},
		})
		return
	}

	// 查询这些池子的基础信息
	var pools []model.LiquidityPool
	if err := ctx.Ctx.DB.Model(&model.LiquidityPool{}).
		Where("pool_address IN (?) AND is_active = ?", poolAddresses, true).
		Order("created_at DESC").
		Find(&pools).Error; err != nil {
		result.Error(c, result.DBQueryFailed)
		return
	}

	// 组装返回数据：poolPair 与 24hVolume（美元格式化）
	items := make([]gin.H, 0, len(pools))
	for _, p := range pools {
		volUSD := compute24hVolumeUSD(p)
		pair := fmt.Sprintf("%s/%s", p.Token0Symbol, p.Token1Symbol)
		items = append(items, gin.H{
			"poolPair":  pair,
			"24hVolume": formatUSD(volUSD),
		})
	}

	result.OK(c, gin.H{
		"poolPerformance": items,
	})
}

// PostLiquidityPools 按照需求返回池子列表（支持 all/my），并计算 24hVolume、24hFees、APY
type LiquidityPoolsRequest struct {
	WalletAddress string `json:"walletAddress" binding:"required"`
	Page          int    `json:"page"`
	PageSize      int    `json:"pageSize"`
	PoolType      string `json:"poolType"` // all 或 my，默认 all
}

func (lp *LiquidityPoolApi) PostLiquidityPools(c *gin.Context) {
	var req LiquidityPoolsRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.WalletAddress == "" {
		result.Error(c, result.InvalidParameter)
		return
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}
	if req.PoolType == "" {
		req.PoolType = "all"
	}

	offset := (req.Page - 1) * req.PageSize

	var pools []model.LiquidityPool
	var total int64

	if req.PoolType == "my" {
		// 查询用户参与过的流动性池地址（去重）
		var poolAddresses []string
		subQuery := ctx.Ctx.DB.Model(&model.LiquidityPoolEvent{}).
			Select("DISTINCT pool_address").
			Where("user_address = ?", req.WalletAddress)

		if err := subQuery.Pluck("pool_address", &poolAddresses).Error; err != nil {
			result.Error(c, result.DBQueryFailed)
			return
		}

		if len(poolAddresses) == 0 {
			result.OK(c, gin.H{
				"total": 0,
				"list":  []gin.H{},
			})
			return
		}

		query := ctx.Ctx.DB.Model(&model.LiquidityPool{}).
			Where("pool_address IN (?) AND is_active = ?", poolAddresses, true)

		if err := query.Count(&total).Error; err != nil {
			result.Error(c, result.DBQueryFailed)
			return
		}
		if err := query.Offset(offset).Limit(req.PageSize).Order("created_at DESC").Find(&pools).Error; err != nil {
			result.Error(c, result.DBQueryFailed)
			return
		}
	} else {
		// all：查询所有活跃池子
		query := ctx.Ctx.DB.Model(&model.LiquidityPool{}).Where("is_active = ?", true)
		if err := query.Count(&total).Error; err != nil {
			result.Error(c, result.DBQueryFailed)
			return
		}
		if err := query.Offset(offset).Limit(req.PageSize).Order("created_at DESC").Find(&pools).Error; err != nil {
			result.Error(c, result.DBQueryFailed)
			return
		}
	}

	// 组装返回并计算统计
	items := make([]gin.H, 0, len(pools))
	for _, p := range pools {
		volUSD := compute24hVolumeUSD(p)
		feesUSD := volUSD * 0.003 // 默认手续费率 0.3%
		apy := computeAPY(p, feesUSD)

		name := fmt.Sprintf("%s/%s", p.Token0Symbol, p.Token1Symbol)
		items = append(items, gin.H{
			"poolId":    fmt.Sprintf("%d", p.Id),
			"poolName":  name,
			"icon":      "https://example.com", // 可后续替换为真实图标地址
			"apy":       apy,
			"24hVolume": formatUSD(volUSD),
			"24hFees":   formatUSD(feesUSD),
		})
	}

	result.OK(c, gin.H{
		"total": total,
		"list":  items,
	})
}

func isStable(symbol string) bool {
	switch symbol {
	case "USDC", "USDT", "DAI":
		return true
	default:
		return false
	}
}

func parseBigInt(s string) *big.Int {
	if s == "" {
		return big.NewInt(0)
	}
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return big.NewInt(0)
	}
	return v
}

func toFloatWithDecimals(v *big.Int, decimals int) float64 {
	if v == nil {
		return 0
	}
	// v / 10^decimals
	f, _ := new(big.Rat).SetFrac(v, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)).Float64()
	return f
}

// compute24hVolumeUSD 仅在稳定币池中返回 USD 交易量，否则为 0
func compute24hVolumeUSD(pool model.LiquidityPool) float64 {
	since := time.Now().Add(-24 * time.Hour)
	var events []model.LiquidityPoolEvent
	if err := ctx.Ctx.DB.Model(&model.LiquidityPoolEvent{}).
		Where("chain_id = ? AND pool_address = ? AND event_type = ? AND created_at >= ?",
			pool.ChainId, pool.PoolAddress, "Swap", since).
		Order("created_at DESC").
		Find(&events).Error; err != nil {
		return 0
	}

	var total float64
	token0Stable := isStable(pool.Token0Symbol)
	token1Stable := isStable(pool.Token1Symbol)

	for _, e := range events {
		a0in := parseBigInt(e.Amount0In)
		a0out := parseBigInt(e.Amount0Out)
		a1in := parseBigInt(e.Amount1In)
		a1out := parseBigInt(e.Amount1Out)

		if token0Stable {
			vol0 := new(big.Int).Add(a0in, a0out)
			total += toFloatWithDecimals(vol0, pool.Token0Decimals)
		} else if token1Stable {
			vol1 := new(big.Int).Add(a1in, a1out)
			total += toFloatWithDecimals(vol1, pool.Token1Decimals)
		}
	}
	return total
}

// computeAPY 基于稳定币侧的 TVL 估算 APY
func computeAPY(pool model.LiquidityPool, feesUSD24h float64) string {
	var tvlUSD float64
	if isStable(pool.Token0Symbol) {
		tvlUSD = 2 * toFloatWithDecimals(parseBigInt(pool.Reserve0), pool.Token0Decimals)
	} else if isStable(pool.Token1Symbol) {
		tvlUSD = 2 * toFloatWithDecimals(parseBigInt(pool.Reserve1), pool.Token1Decimals)
	}

	if tvlUSD <= 0 {
		return "-"
	}
	apy := (feesUSD24h / tvlUSD) * 365 * 100
	if math.IsNaN(apy) || math.IsInf(apy, 0) {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", apy)
}

func formatUSD(v float64) string {
	if v <= 0 {
		return "$0"
	}
	// K/M/B 格式化
	if v >= 1_000_000_000 {
		return fmt.Sprintf("$%.1fB", v/1_000_000_000)
	}
	if v >= 1_000_000 {
		return fmt.Sprintf("$%.1fM", v/1_000_000)
	}
	if v >= 1_000 {
		return fmt.Sprintf("$%.1fK", v/1_000)
	}
	return fmt.Sprintf("$%.2f", v)
}
