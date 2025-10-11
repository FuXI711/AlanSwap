package core

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mumu/cryptoSwap/src/app/sync"
	"github.com/mumu/cryptoSwap/src/core/chainclient"
	"github.com/mumu/cryptoSwap/src/core/config"
	"github.com/mumu/cryptoSwap/src/core/ctx"
	"github.com/mumu/cryptoSwap/src/core/db"
	"github.com/mumu/cryptoSwap/src/core/gin/router"
	"github.com/mumu/cryptoSwap/src/core/log"
	"go.uber.org/zap"
)

// Start
//
//	@Description:
//	@param configFile
//	@param serverType  为了简单区分不同的服务类型，1 代表 api服务  2 代表监听服务
func Start(configFile string, serverType int) {
	c, cancel := context.WithCancel(context.Background())
	defer cancel()
	// 初始化配置信息
	initConfig(configFile)
	// 初始化日志组件
	initLog()
	// 启用性能监控组件
	initPprof()
	// 初始化数据库/Redis
	initDB()
	// 初始化区块链客户端
	initChainClient()

	if serverType == 1 {
		initApiGin()
	} else if serverType == 2 {
		// 初始化Gin
		initGin()
		//开启线程获取scan log
		initSync(c)
		//计算积分
		initComputeIntegral()
	}
}
func initConfig(configFile string) {
	ctx.Ctx.Config = config.InitConfig(configFile)
}
func initPprof() {
	if !config.Conf.Monitor.PprofEnable {
		return
	}
	log.Logger.Info("init pprof")
	go func() {
		err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", config.Conf.Monitor.PprofPort), nil)
		if err != nil {
			log.Logger.Error("init pprof error", zap.Error(err))
			return
		}
	}()
}
func initLog() {
	ctx.Ctx.Log = log.InitLog()
}
func initDB() {
	ctx.Ctx.DB = db.InitPgsql()
	ctx.Ctx.Redis = db.InitRedis()
}
func initChainClient() {
	chainMap := make(map[int]*chainclient.ChainClient)
	for _, chain := range config.Conf.Chains {
		log.Logger.Info("正在初始化链客户端", zap.Int("chain_id", chain.ChainId), zap.String("endpoint", chain.Endpoint))
		client, err := chainclient.New(chain.ChainId, chain.Endpoint)
		if err != nil {
			log.Logger.Error("init chain client error", zap.Error(err))
			panic(err)
		}

		chainMap[chain.ChainId] = &client
		log.Logger.Info("链客户端初始化成功", zap.Int("chain_id", chain.ChainId))
	}

	ctx.Ctx.ChainMap = chainMap
}
func initGin() {
	r := router.InitRouter()
	ctx.Ctx.Gin = r
	router.Bind(r, &ctx.Ctx)
	// 在goroutine中启动服务器，避免阻塞主线程
	//initGin()函数中的r.Run()是阻塞调用，会阻止后续代码执行
	go func() {
		err := r.Run(":" + ctx.Ctx.Config.App.Port)
		if err != nil {
			panic(err)
		}
	}()

	// 给服务器一点时间启动
	time.Sleep(100 * time.Millisecond)
}

func initApiGin() {
	r := router.InitRouter()
	ctx.Ctx.Gin = r
	router.ApiBind(r, &ctx.Ctx)
	err := r.Run(":" + ctx.Ctx.Config.App.APIPort)
	if err != nil {
		panic(err)
	}
}

func initSync(c context.Context) {
	go sync.StartSync(c)
}
func initComputeIntegral() {
	go sync.InitComputeIntegral()
}
