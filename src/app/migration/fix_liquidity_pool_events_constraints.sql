-- 修复liquidity_pool_events表的约束问题 (PostgreSQL版本)
-- 移除token0_address和token1_address字段的NOT NULL约束

-- 修改token0_address字段，移除NOT NULL约束
ALTER TABLE liquidity_pool_events ALTER COLUMN token0_address DROP NOT NULL;

-- 修改token1_address字段，移除NOT NULL约束  
ALTER TABLE liquidity_pool_events ALTER COLUMN token1_address DROP NOT NULL;

-- 检查并清理可能存在的空字符串数据
-- 如果字段是numeric类型，空字符串会导致错误，所以我们需要先处理这些数据

-- 方法1：删除包含无效数据的记录（如果数据不重要）
-- DELETE FROM liquidity_pool_events WHERE reserve0 = '' OR reserve1 = '' OR price = '' OR liquidity = '';

-- 方法2：或者先将这些字段设为NULL，然后更新为'0'
UPDATE liquidity_pool_events 
SET reserve0 = NULL 
WHERE reserve0 = '';

UPDATE liquidity_pool_events 
SET reserve1 = NULL 
WHERE reserve1 = '';

UPDATE liquidity_pool_events 
SET price = NULL 
WHERE price = '';

UPDATE liquidity_pool_events 
SET liquidity = NULL 
WHERE liquidity = '';

-- 现在为NULL值设置默认值
UPDATE liquidity_pool_events 
SET reserve0 = '0' 
WHERE reserve0 IS NULL;

UPDATE liquidity_pool_events 
SET reserve1 = '0' 
WHERE reserve1 IS NULL;

UPDATE liquidity_pool_events 
SET price = '0' 
WHERE price IS NULL;

UPDATE liquidity_pool_events 
SET liquidity = '0' 
WHERE liquidity IS NULL;

-- 为decimal类型字段设置默认值约束（PostgreSQL语法）
ALTER TABLE liquidity_pool_events ALTER COLUMN reserve0 SET DEFAULT '0';
ALTER TABLE liquidity_pool_events ALTER COLUMN reserve1 SET DEFAULT '0';
ALTER TABLE liquidity_pool_events ALTER COLUMN price SET DEFAULT '0';
ALTER TABLE liquidity_pool_events ALTER COLUMN liquidity SET DEFAULT '0';

-- 添加字段注释
COMMENT ON COLUMN liquidity_pool_events.token0_address IS 'Token0地址，暂时可为空';
COMMENT ON COLUMN liquidity_pool_events.token1_address IS 'Token1地址，暂时可为空';
COMMENT ON COLUMN liquidity_pool_events.reserve0 IS '池子Token0储备量';
COMMENT ON COLUMN liquidity_pool_events.reserve1 IS '池子Token1储备量';
COMMENT ON COLUMN liquidity_pool_events.price IS '价格';
COMMENT ON COLUMN liquidity_pool_events.liquidity IS '流动性';
