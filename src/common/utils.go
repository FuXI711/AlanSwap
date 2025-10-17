package common

import (
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
)

// GetCurrentAbPath 获取当前项目绝对路径
func GetCurrentAbPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}

	// 获取当前文件所在目录
	dir := filepath.Dir(filename)

	// 获取上两级目录
	abPath := filepath.Join(dir, "..", "..")

	// Clean 会清理多余的 ../ 和 . 等符号，确保路径合法
	clean := filepath.Clean(abPath)
	return clean
}

// 从liquidity_pool_api.go复用的辅助函数
func ParseChainId(chainIdStr string) (int64, bool) {
	if chainIdStr == "" {
		return 0, true
	}
	id, err := strconv.ParseInt(chainIdStr, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

func ValidateHexAddress(addr string) bool {
	return addr != "" && common.IsHexAddress(addr)
}

func GetConfigAbPath() string {
	return GetCurrentAbPath() + "/config"
}
