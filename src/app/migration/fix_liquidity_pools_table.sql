-- 修复liquidity_pools表的约束问题
-- 移除token0_address和token1_address字段的NOT NULL约束

-- 移除NOT NULL约束
ALTER TABLE liquidity_pools ALTER COLUMN token0_address DROP NOT NULL;
ALTER TABLE liquidity_pools ALTER COLUMN token1_address DROP NOT NULL;

-- 添加字段注释
COMMENT ON COLUMN liquidity_pools.token0_address IS 'Token0地址，暂时可为空';
COMMENT ON COLUMN liquidity_pools.token1_address IS 'Token1地址，暂时可为空';

-- 检查是否有包含空字符串的decimal字段数据
-- 如果有，建议手动删除或修复
-- SELECT * FROM liquidity_pools WHERE reserve0 = '' OR reserve1 = '' OR total_supply = '' OR price = '' OR volume_24h = '';

-- 如果发现问题数据，可以手动删除：
-- DELETE FROM liquidity_pools WHERE reserve0 = '' OR reserve1 = '' OR total_supply = '' OR price = '' OR volume_24h = '';
