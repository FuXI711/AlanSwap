-- 为chain表添加service_type字段用于区分服务类型
ALTER TABLE chain ADD COLUMN service_type VARCHAR(20);

-- 添加字段注释
COMMENT ON COLUMN chain.service_type IS '服务类型: staking(质押池)/liquidity(流动性池)';

-- 为现有数据设置默认值（根据实际情况修改）
-- 示例：为现有链配置添加服务类型
-- UPDATE chain SET service_type = 'staking' WHERE address IS NOT NULL AND address != '';
-- UPDATE chain SET service_type = 'liquidity' WHERE dex_address IS NOT NULL AND dex_address != '';

-- 为同一个链创建不同的服务配置示例（注释掉，根据需要启用）
/*
-- 为链ID 11155111 创建质押池配置
INSERT INTO chain (chain_id, chain_name, address, service_type, last_block_num) 
VALUES (11155111, 'sepolia-staking', '0x质押池合约地址', 'staking', 0);

-- 为链ID 11155111 创建流动性池配置  
INSERT INTO chain (chain_id, chain_name, dex_address, service_type, last_block_num)
VALUES (11155111, 'sepolia-liquidity', '0x流动性池合约地址', 'liquidity', 0);
*/

-- 添加复合索引提高查询性能
CREATE INDEX idx_chain_service_type ON chain(chain_id, service_type);
