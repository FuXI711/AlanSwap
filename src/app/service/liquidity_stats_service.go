package service

import (
	"fmt"
	"math/big"
	"time"

	"github.com/mumu/cryptoSwap/src/app/model"
	"github.com/mumu/cryptoSwap/src/core/ctx"
)

type LiquidityStatsService struct {
	req *model.LiquidityStatsRequest
}

func NewLiquidityStatsService() *LiquidityStatsService {
	return &LiquidityStatsService{req: &model.LiquidityStatsRequest{}}
}

// GetLiquidityStats 获取流动性统计信息
func (s *LiquidityStatsService) GetLiquidityStats(req *model.LiquidityStatsRequest) (*model.LiquidityStatsResponse, error) {
	stats := &model.LiquidityStatsResponse{}

	// 获取我的流动性价值
	myLiquidityValue, err := s.calculateMyLiquidityValue(req.UserAddress, req.ChainId)
	if err != nil {
		return nil, err
	}
	stats.MyLiquidityValue = fmt.Sprintf("$%.2f", myLiquidityValue)

	// 获取我的流动性变化率
	periodChange, err := s.calculateMyLiquidityPeriodChange(req.UserAddress, req.ChainId, req.Period)
	if err != nil {
		return nil, err
	}
	stats.MyLiquidityPeriodChange = fmt.Sprintf("%+.1f%%", periodChange*100)

	// 获取累计手续费
	totalFees, err := s.calculateTotalFees(req.ChainId)
	if err != nil {
		return nil, err
	}
	stats.TotalFees = fmt.Sprintf("$%.2f", totalFees)

	// 获取今日手续费变化量
	feesTodayChange, err := s.calculateFeesTodayChange(req.ChainId)
	if err != nil {
		return nil, err
	}
	stats.TotalFeesTodayChange = fmt.Sprintf("+$%.2f", feesTodayChange)

	// 获取活跃池子数量
	activePoolsCount, err := s.getActivePoolsCount(req.ChainId)
	if err != nil {
		return nil, err
	}
	stats.ActivePoolsCount = activePoolsCount

	// 获取总池子数量
	totalPoolsCount, err := s.getTotalPoolsCount(req.ChainId)
	if err != nil {
		return nil, err
	}
	stats.TotalPoolsCount = totalPoolsCount

	return stats, nil
}

// calculateMyLiquidityValue 计算我的流动性总价值
func (s *LiquidityStatsService) calculateMyLiquidityValue(userAddress string, chainId int64) (float64, error) {
	var totalValue float64

	// 查询用户参与的流动性池事件
	var userEvents []model.LiquidityPoolEvent
	query := ctx.Ctx.DB.Where("user_address = ?", userAddress)
	if chainId > 0 {
		query = query.Where("chain_id = ?", chainId)
	}

	err := query.Find(&userEvents).Error
	if err != nil {
		return 0, err
	}

	// 计算每个池子的流动性价值
	poolValues := make(map[string]float64)
	for _, event := range userEvents {
		// 获取池子当前价格
		var pool model.LiquidityPool
		err := ctx.Ctx.DB.Where("pool_address = ? AND chain_id = ?", event.PoolAddress, event.ChainId).
			First(&pool).Error
		if err != nil {
			continue
		}

		// 计算用户在该池子的LP代币价值
		price, _ := new(big.Float).SetString(pool.Price)
		if price == nil {
			continue
		}

		// 简化计算：假设用户LP代币价值与池子总价值成比例
		// 实际应该根据用户LP代币数量计算
		poolValue, _ := price.Float64()
		poolValues[event.PoolAddress] = poolValue
	}

	// 累加所有池子的价值
	for _, value := range poolValues {
		totalValue += value
	}

	return totalValue, nil
}

// calculateMyLiquidityPeriodChange 计算我的流动性变化率
func (s *LiquidityStatsService) calculateMyLiquidityPeriodChange(userAddress string, chainId int64, period string) (float64, error) {
	// 获取当前流动性价值
	currentValue, err := s.calculateMyLiquidityValue(userAddress, chainId)
	if err != nil {
		return 0, err
	}

	// 获取周期开始时的流动性价值
	startTime := s.getPeriodStartTime(period)
	previousValue, err := s.getHistoricalLiquidityValue(userAddress, chainId, startTime)
	if err != nil {
		return 0, err
	}

	if previousValue == 0 {
		return 0, nil
	}

	return (currentValue - previousValue) / previousValue, nil
}

// calculateTotalFees 计算累计手续费
func (s *LiquidityStatsService) calculateTotalFees(chainId int64) (float64, error) {
	var totalFees float64

	// 查询所有Swap事件，计算手续费
	// 假设手续费为交易量的0.3%
	var swapEvents []model.LiquidityPoolEvent
	query := ctx.Ctx.DB.Where("event_type = ?", "Swap")
	if chainId > 0 {
		query = query.Where("chain_id = ?", chainId)
	}

	err := query.Find(&swapEvents).Error
	if err != nil {
		return 0, err
	}

	for _, event := range swapEvents {
		// 计算交易量（取较大的输入或输出）
		amount0In, _ := new(big.Float).SetString(event.Amount0In)
		amount1In, _ := new(big.Float).SetString(event.Amount1In)
		amount0Out, _ := new(big.Float).SetString(event.Amount0Out)
		amount1Out, _ := new(big.Float).SetString(event.Amount1Out)

		var volume *big.Float
		if amount0In != nil && amount0In.Sign() > 0 {
			volume = amount0In
		} else if amount1In != nil && amount1In.Sign() > 0 {
			volume = amount1In
		} else if amount0Out != nil && amount0Out.Sign() > 0 {
			volume = amount0Out
		} else if amount1Out != nil && amount1Out.Sign() > 0 {
			volume = amount1Out
		}

		if volume != nil {
			// 手续费率为0.3%
			fee := new(big.Float).Mul(volume, big.NewFloat(0.003))
			feeFloat, _ := fee.Float64()
			totalFees += feeFloat
		}
	}

	return totalFees, nil
}

// calculateFeesTodayChange 计算今日手续费变化量
func (s *LiquidityStatsService) calculateFeesTodayChange(chainId int64) (float64, error) {
	// 获取今日手续费
	var todayFees float64
	query := ctx.Ctx.DB.Where("event_type = ? AND DATE(created_at) = CURDATE()", "Swap")
	if chainId > 0 {
		query = query.Where("chain_id = ?", chainId)
	}

	var todayEvents []model.LiquidityPoolEvent
	err := query.Find(&todayEvents).Error
	if err != nil {
		return 0, err
	}

	for _, event := range todayEvents {
		amount0In, _ := new(big.Float).SetString(event.Amount0In)
		if amount0In != nil {
			fee := new(big.Float).Mul(amount0In, big.NewFloat(0.003))
			feeFloat, _ := fee.Float64()
			todayFees += feeFloat
		}
	}

	return todayFees, nil
}

// getActivePoolsCount 获取活跃池子数量
func (s *LiquidityStatsService) getActivePoolsCount(chainId int64) (int, error) {
	var count int64
	query := ctx.Ctx.DB.Model(&model.LiquidityPool{}).Where("is_active = ?", true)
	if chainId > 0 {
		query = query.Where("chain_id = ?", chainId)
	}

	err := query.Count(&count).Error
	return int(count), err
}

// getTotalPoolsCount 获取总池子数量
func (s *LiquidityStatsService) getTotalPoolsCount(chainId int64) (int, error) {
	var count int64
	query := ctx.Ctx.DB.Model(&model.LiquidityPool{})
	if chainId > 0 {
		query = query.Where("chain_id = ?", chainId)
	}

	err := query.Count(&count).Error
	return int(count), err
}

// getPeriodStartTime 获取周期开始时间
func (s *LiquidityStatsService) getPeriodStartTime(period string) time.Time {
	now := time.Now()
	switch period {
	case "today":
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "week":
		return now.AddDate(0, 0, -7)
	case "month":
		return now.AddDate(0, -1, 0)
	default:
		return time.Time{} // 所有时间
	}
}

// getHistoricalLiquidityValue 获取历史流动性价值
func (s *LiquidityStatsService) getHistoricalLiquidityValue(userAddress string, chainId int64, startTime time.Time) (float64, error) {
	// 简化实现：查询指定时间点之后的用户事件
	var historicalValue float64

	var userEvents []model.LiquidityPoolEvent
	query := ctx.Ctx.DB.Where("user_address = ? AND created_at >= ?", userAddress, startTime)
	if chainId > 0 {
		query = query.Where("chain_id = ?", chainId)
	}

	err := query.Find(&userEvents).Error
	if err != nil {
		return 0, err
	}

	// 简化计算：使用事件发生时的价格估算历史价值
	for _, event := range userEvents {
		price, _ := new(big.Float).SetString(event.Price)
		if price != nil {
			priceFloat, _ := price.Float64()
			historicalValue += priceFloat
		}
	}

	return historicalValue, nil
}
