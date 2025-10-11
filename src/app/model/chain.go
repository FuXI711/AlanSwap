package model

type Chain struct {
	Id                    int64  `json:"id" gorm:"column:id;primaryKey"`
	ChainId               int64  `json:"chainId" gorm:"column:chain_id"`
	ChainName             string `json:"chainName" gorm:"column:chain_name"`
	Address               string `json:"address" gorm:"column:address"`
	DexAddress            string `json:"dexAddress" gorm:"column:dex_address"`
	LastBlockNum          uint64 `json:"lastBlockNum" gorm:"column:last_block_num"`                    // 通用区块号（保留兼容性）
	StakingLastBlockNum   uint64 `json:"stakingLastBlockNum" gorm:"column:staking_last_block_num"`     // 质押池监听区块号
	LiquidityLastBlockNum uint64 `json:"liquidityLastBlockNum" gorm:"column:liquidity_last_block_num"` // 流动性池监听区块号
}

// TableName 指定表名
func (Chain) TableName() string {
	return "chain"
}
