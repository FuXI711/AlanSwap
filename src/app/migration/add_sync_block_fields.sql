-- 为chain表添加独立的监听区块号字段
ALTER TABLE chain ADD COLUMN IF NOT EXISTS staking_last_block_num BIGINT DEFAULT 0;
ALTER TABLE chain ADD COLUMN IF NOT EXISTS liquidity_last_block_num BIGINT DEFAULT 0;

-- 为现有数据设置初始值（将原有的last_block_num复制到新字段）
UPDATE chain SET 
    staking_last_block_num = last_block_num,
    liquidity_last_block_num = last_block_num
WHERE staking_last_block_num = 0 AND liquidity_last_block_num = 0;

-- 添加字段注释
COMMENT ON COLUMN chain.staking_last_block_num IS '质押池监听服务的最后同步区块号';
COMMENT ON COLUMN chain.liquidity_last_block_num IS '流动性池监听服务的最后同步区块号';

-- 可选：为现有链设置初始的DEX合约地址（根据实际情况修改）
-- UPDATE chain SET dex_address = '0x72e46e15ef83c896de44B1874B4AF7dDAB5b4F74' WHERE chain_id = 11155111 AND dex_address IS NULL;
