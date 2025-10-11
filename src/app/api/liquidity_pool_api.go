package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mumu/cryptoSwap/src/app/model"
	"github.com/mumu/cryptoSwap/src/core/ctx"
	"github.com/mumu/cryptoSwap/src/core/result"
)

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
func GetLiquidityPoolEvents(c *gin.Context) {
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
func GetLiquidityPoolStats(c *gin.Context) {
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
		"totalPools":  totalPools,
		"totalEvents": totalEvents,
		"todayEvents": todayEvents,
	})
}
