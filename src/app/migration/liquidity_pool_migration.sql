-- 流动性池事件表
CREATE TABLE IF NOT EXISTS liquidity_pool_events (
    id BIGSERIAL PRIMARY KEY,
    chain_id BIGINT NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    block_number BIGINT NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    pool_address VARCHAR(42) NOT NULL,
    token0_address VARCHAR(42) NOT NULL,
    token1_address VARCHAR(42) NOT NULL,
    user_address VARCHAR(42) NOT NULL,
    amount0_in DECIMAL(78,0) DEFAULT '0',
    amount1_in DECIMAL(78,0) DEFAULT '0',
    amount0_out DECIMAL(78,0) DEFAULT '0',
    amount1_out DECIMAL(78,0) DEFAULT '0',
    reserve0 DECIMAL(78,0) DEFAULT '0',
    reserve1 DECIMAL(78,0) DEFAULT '0',
    price DECIMAL(30,18) DEFAULT '0',
    liquidity DECIMAL(78,0) DEFAULT '0',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 流动性池信息表
CREATE TABLE IF NOT EXISTS liquidity_pools (
    id BIGSERIAL PRIMARY KEY,
    chain_id BIGINT NOT NULL,
    pool_address VARCHAR(42) NOT NULL,
    token0_address VARCHAR(42) NOT NULL,
    token1_address VARCHAR(42) NOT NULL,
    token0_symbol VARCHAR(20),
    token1_symbol VARCHAR(20),
    token0_decimals INTEGER DEFAULT 18,
    token1_decimals INTEGER DEFAULT 18,
    reserve0 DECIMAL(78,0) DEFAULT '0',
    reserve1 DECIMAL(78,0) DEFAULT '0',
    total_supply DECIMAL(78,0) DEFAULT '0',
    price DECIMAL(30,18) DEFAULT '0',
    volume_24h DECIMAL(78,0) DEFAULT '0',
    tx_count BIGINT DEFAULT 0,
    last_block_num BIGINT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(chain_id, pool_address)
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_liquidity_pool_events_chain_id ON liquidity_pool_events(chain_id);
CREATE INDEX IF NOT EXISTS idx_liquidity_pool_events_tx_hash ON liquidity_pool_events(tx_hash);
CREATE INDEX IF NOT EXISTS idx_liquidity_pool_events_pool_address ON liquidity_pool_events(pool_address);
CREATE INDEX IF NOT EXISTS idx_liquidity_pool_events_user_address ON liquidity_pool_events(user_address);
CREATE INDEX IF NOT EXISTS idx_liquidity_pool_events_event_type ON liquidity_pool_events(event_type);
CREATE INDEX IF NOT EXISTS idx_liquidity_pool_events_created_at ON liquidity_pool_events(created_at);

CREATE INDEX IF NOT EXISTS idx_liquidity_pools_chain_id ON liquidity_pools(chain_id);
CREATE INDEX IF NOT EXISTS idx_liquidity_pools_pool_address ON liquidity_pools(pool_address);
CREATE INDEX IF NOT EXISTS idx_liquidity_pools_is_active ON liquidity_pools(is_active);
CREATE INDEX IF NOT EXISTS idx_liquidity_pools_tx_count ON liquidity_pools(tx_count);

-- 添加注释
COMMENT ON TABLE liquidity_pool_events IS '流动性池事件记录表';
COMMENT ON TABLE liquidity_pools IS '流动性池信息表';

COMMENT ON COLUMN liquidity_pool_events.chain_id IS '链ID';
COMMENT ON COLUMN liquidity_pool_events.tx_hash IS '交易哈希';
COMMENT ON COLUMN liquidity_pool_events.block_number IS '区块号';
COMMENT ON COLUMN liquidity_pool_events.event_type IS '事件类型：Swap, AddLiquidity, RemoveLiquidity';
COMMENT ON COLUMN liquidity_pool_events.pool_address IS '流动性池合约地址';
COMMENT ON COLUMN liquidity_pool_events.token0_address IS '代币0地址';
COMMENT ON COLUMN liquidity_pool_events.token1_address IS '代币1地址';
COMMENT ON COLUMN liquidity_pool_events.user_address IS '用户地址';
COMMENT ON COLUMN liquidity_pool_events.amount0_in IS '代币0输入数量';
COMMENT ON COLUMN liquidity_pool_events.amount1_in IS '代币1输入数量';
COMMENT ON COLUMN liquidity_pool_events.amount0_out IS '代币0输出数量';
COMMENT ON COLUMN liquidity_pool_events.amount1_out IS '代币1输出数量';

COMMENT ON COLUMN liquidity_pools.chain_id IS '链ID';
COMMENT ON COLUMN liquidity_pools.pool_address IS '流动性池合约地址';
COMMENT ON COLUMN liquidity_pools.token0_address IS '代币0地址';
COMMENT ON COLUMN liquidity_pools.token1_address IS '代币1地址';
COMMENT ON COLUMN liquidity_pools.token0_symbol IS '代币0符号';
COMMENT ON COLUMN liquidity_pools.token1_symbol IS '代币1符号';
COMMENT ON COLUMN liquidity_pools.token0_decimals IS '代币0精度';
COMMENT ON COLUMN liquidity_pools.token1_decimals IS '代币1精度';
COMMENT ON COLUMN liquidity_pools.reserve0 IS '代币0储备量';
COMMENT ON COLUMN liquidity_pools.reserve1 IS '代币1储备量';
COMMENT ON COLUMN liquidity_pools.total_supply IS '总供应量';
COMMENT ON COLUMN liquidity_pools.price IS '价格';
COMMENT ON COLUMN liquidity_pools.volume_24h IS '24小时交易量';
COMMENT ON COLUMN liquidity_pools.tx_count IS '交易次数';
COMMENT ON COLUMN liquidity_pools.last_block_num IS '最后处理的区块号';
COMMENT ON COLUMN liquidity_pools.is_active IS '是否活跃';
