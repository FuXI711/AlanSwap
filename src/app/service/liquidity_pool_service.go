package service

import (
	"github.com/mumu/cryptoSwap/src/app/model"
	"github.com/mumu/cryptoSwap/src/core/ctx"
)

type LiquidityPoolService struct{}

// GetPoolByAddress 根据池子地址获取流动性池信息
func (s *LiquidityPoolService) GetPoolByAddress(chainId int64, poolAddress string) (*model.LiquidityPool, error) {
	var pool model.LiquidityPool
	err := ctx.Ctx.DB.Where("chain_id = ? AND pool_address = ?", chainId, poolAddress).First(&pool).Error
	if err != nil {
		return nil, err
	}
	return &pool, nil
}

// GetPoolEvents 获取流动性池事件
func (s *LiquidityPoolService) GetPoolEvents(chainId int64, poolAddress string, eventType string, limit int) ([]model.LiquidityPoolEvent, error) {
	var events []model.LiquidityPoolEvent
	query := ctx.Ctx.DB.Where("chain_id = ? AND pool_address = ?", chainId, poolAddress)

	if eventType != "" {
		query = query.Where("event_type = ?", eventType)
	}

	err := query.Order("created_at DESC").Limit(limit).Find(&events).Error
	return events, err
}

// GetUserEvents 获取用户相关的事件
func (s *LiquidityPoolService) GetUserEvents(chainId int64, userAddress string, limit int) ([]model.LiquidityPoolEvent, error) {
	var events []model.LiquidityPoolEvent
	err := ctx.Ctx.DB.Where("chain_id = ? AND user_address = ?", chainId, userAddress).
		Order("created_at DESC").Limit(limit).Find(&events).Error
	return events, err
}

// CreatePool 创建新的流动性池记录
func (s *LiquidityPoolService) CreatePool(pool *model.LiquidityPool) error {
	return ctx.Ctx.DB.Create(pool).Error
}

// UpdatePool 更新流动性池信息
func (s *LiquidityPoolService) UpdatePool(pool *model.LiquidityPool) error {
	return ctx.Ctx.DB.Save(pool).Error
}

// BatchCreateEvents 批量创建事件记录
func (s *LiquidityPoolService) BatchCreateEvents(events []model.LiquidityPoolEvent) error {
	return ctx.Ctx.DB.CreateInBatches(events, 100).Error
}

// GetPoolStats 获取流动性池统计信息
func (s *LiquidityPoolService) GetPoolStats(chainId int64) (map[string]interface{}, error) {
	var stats map[string]interface{} = make(map[string]interface{})

	// 总池子数
	var totalPools int64
	err := ctx.Ctx.DB.Model(&model.LiquidityPool{}).Where("chain_id = ? AND is_active = ?", chainId, true).Count(&totalPools).Error
	if err != nil {
		return nil, err
	}
	stats["totalPools"] = totalPools

	// 总事件数
	var totalEvents int64
	err = ctx.Ctx.DB.Model(&model.LiquidityPoolEvent{}).Where("chain_id = ?", chainId).Count(&totalEvents).Error
	if err != nil {
		return nil, err
	}
	stats["totalEvents"] = totalEvents

	// 今日事件数
	var todayEvents int64
	err = ctx.Ctx.DB.Model(&model.LiquidityPoolEvent{}).Where("chain_id = ? AND DATE(created_at) = CURDATE()", chainId).Count(&todayEvents).Error
	if err != nil {
		return nil, err
	}
	stats["todayEvents"] = todayEvents

	// 活跃用户数
	var activeUsers int64
	err = ctx.Ctx.DB.Model(&model.LiquidityPoolEvent{}).Where("chain_id = ?", chainId).
		Distinct("user_address").Count(&activeUsers).Error
	if err != nil {
		return nil, err
	}
	stats["activeUsers"] = activeUsers

	return stats, nil
}

// GetTopPools 获取交易量最大的流动性池
func (s *LiquidityPoolService) GetTopPools(chainId int64, limit int) ([]model.LiquidityPool, error) {
	var pools []model.LiquidityPool
	err := ctx.Ctx.DB.Where("chain_id = ? AND is_active = ?", chainId, true).
		Order("tx_count DESC").Limit(limit).Find(&pools).Error
	return pools, err
}

// GetRecentEvents 获取最近的事件
func (s *LiquidityPoolService) GetRecentEvents(chainId int64, limit int) ([]model.LiquidityPoolEvent, error) {
	var events []model.LiquidityPoolEvent
	err := ctx.Ctx.DB.Where("chain_id = ?", chainId).
		Order("created_at DESC").Limit(limit).Find(&events).Error
	return events, err
}

// GetPoolVolume 获取池子的交易量统计
func (s *LiquidityPoolService) GetPoolVolume(chainId int64, poolAddress string, days int) (map[string]interface{}, error) {
	var stats map[string]interface{} = make(map[string]interface{})

	// 总交易量
	var totalVolume int64
	err := ctx.Ctx.DB.Model(&model.LiquidityPoolEvent{}).
		Where("chain_id = ? AND pool_address = ? AND event_type = ?", chainId, poolAddress, "Swap").
		Count(&totalVolume).Error
	if err != nil {
		return nil, err
	}
	stats["totalVolume"] = totalVolume

	// 指定天数内的交易量
	var periodVolume int64
	err = ctx.Ctx.DB.Model(&model.LiquidityPoolEvent{}).
		Where("chain_id = ? AND pool_address = ? AND event_type = ? AND created_at >= DATE_SUB(NOW(), INTERVAL ? DAY)",
			chainId, poolAddress, "Swap", days).
		Count(&periodVolume).Error
	if err != nil {
		return nil, err
	}
	stats["periodVolume"] = periodVolume

	return stats, nil
}
