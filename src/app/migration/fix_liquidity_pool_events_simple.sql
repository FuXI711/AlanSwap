-- 简单修复liquidity_pool_events表的约束问题
-- 只处理最关键的NOT NULL约束问题

-- 移除token0_address和token1_address字段的NOT NULL约束
ALTER TABLE liquidity_pool_events ALTER COLUMN token0_address DROP NOT NULL;
ALTER TABLE liquidity_pool_events ALTER COLUMN token1_address DROP NOT NULL;

-- 添加字段注释
COMMENT ON COLUMN liquidity_pool_events.token0_address IS 'Token0地址，暂时可为空';
COMMENT ON COLUMN liquidity_pool_events.token1_address IS 'Token1地址，暂时可为空';

-- 如果表中已有包含空字符串的记录，建议手动删除或修复
-- 可以使用以下查询检查：
-- SELECT * FROM liquidity_pool_events WHERE reserve0 = '' OR reserve1 = '' OR price = '' OR liquidity = '';

-- 如果发现问题数据，可以手动删除：
-- DELETE FROM liquidity_pool_events WHERE reserve0 = '' OR reserve1 = '' OR price = '' OR liquidity = '';
