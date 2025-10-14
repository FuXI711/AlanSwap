package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mumu/cryptoSwap/src/app/model"
	"github.com/mumu/cryptoSwap/src/app/service"
)

type LiquidityStatsApi struct {
	svc *service.LiquidityStatsService
}

func NewLiquidityStatsApi() *LiquidityStatsApi {
	return &LiquidityStatsApi{
		svc: service.NewLiquidityStatsService(),
	}
}

// GetLiquidityStats 处理流动性统计请求
func (api *LiquidityStatsApi) GetLiquidityStats(c *gin.Context) {
	// 绑定请求参数
	var req model.LiquidityStatsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 参数验证
	if req.UserAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户地址不能为空"})
		return
	}

	// 初始化服务

	service := service.NewLiquidityStatsService()

	// 获取统计数据
	stats, err := service.GetLiquidityStats(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回响应
	c.JSON(http.StatusOK, stats)
}
