package sync

import (
	"context"
	"sync"

	"github.com/mumu/cryptoSwap/src/core/log"
)

func StartSync(c context.Context) {
	var wg sync.WaitGroup

	// 启动质押池事件监听
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Logger.Info("启动质押池监听服务")
		StartStakingPoolSync(c)
	}()

	// 启动流动性池事件监听
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Logger.Info("启动流动性池监听服务")
		StartLiquidityPoolSync(c)
	}()

	//一直等待
	wg.Wait()
}
