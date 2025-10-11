-- 安全修复liquidity_pool_events表的约束问题
-- 分步骤执行，避免数据类型冲突

-- 步骤1：移除NOT NULL约束
ALTER TABLE liquidity_pool_events ALTER COLUMN token0_address DROP NOT NULL;
ALTER TABLE liquidity_pool_events ALTER COLUMN token1_address DROP NOT NULL;

-- 步骤2：检查现有数据的问题
-- 如果表中已有数据且包含空字符串，我们需要先处理
-- 查看是否有问题的数据
SELECT COUNT(*) FROM liquidity_pool_events 
WHERE reserve0 = '' OR reserve1 = '' OR price = '' OR liquidity = '';

-- 步骤3：如果有问题的数据，可以选择删除或修复
-- 选项A：删除有问题的记录（如果数据不重要）
-- DELETE FROM liquidity_pool_events WHERE reserve0 = '' OR reserve1 = '' OR price = '' OR liquidity = '';

-- 选项B：修复数据（推荐）
-- 先将空字符串设为NULL，然后设为'0'
UPDATE liquidity_pool_events SET reserve0 = NULL WHERE reserve0 = '';
UPDATE liquidity_pool_events SET reserve1 = NULL WHERE reserve1 = '';
UPDATE liquidity_pool_events SET price = NULL WHERE price = '';
UPDATE liquidity_pool_events SET liquidity = NULL WHERE liquidity = '';

-- 然后将NULL值设为'0'
UPDATE liquidity_pool_events SET reserve0 = '0' WHERE reserve0 IS NULL;
UPDATE liquidity_pool_events SET reserve1 = '0' WHERE reserve1 IS NULL;
UPDATE liquidity_pool_events SET price = '0' WHERE price IS NULL;
UPDATE liquidity_pool_events SET liquidity = '0' WHERE liquidity IS NULL;

-- 步骤4：添加字段注释
COMMENT ON COLUMN liquidity_pool_events.token0_address IS 'Token0地址，暂时可为空';
COMMENT ON COLUMN liquidity_pool_events.token1_address IS 'Token1地址，暂时可为空';
COMMENT ON COLUMN liquidity_pool_events.reserve0 IS '池子Token0储备量';
COMMENT ON COLUMN liquidity_pool_events.reserve1 IS '池子Token1储备量';
COMMENT ON COLUMN liquidity_pool_events.price IS '价格';
COMMENT ON COLUMN liquidity_pool_events.liquidity IS '流动性';
