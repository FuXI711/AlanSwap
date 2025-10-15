-- 安全处理：若上一次执行中事务处于 aborted 状态，先回滚以清理环境。
-- 在无事务时该语句只会产生警告，不影响执行。
ROLLBACK;

BEGIN;

-- 空投活动聚合表
-- 数据来源：
--  1) 监听 AirdropCreated / AirdropActivated / MerkleRootUpdated 事件进行基础信息更新；
--  2) 通过读取链上视图 getAirdropInfo(airdropId) 补齐 start_time、end_time、is_active、claimed_reward。
CREATE TABLE IF NOT EXISTS airdrop_activity (
  id BIGSERIAL PRIMARY KEY,
  chain_id INTEGER NOT NULL,
  contract_address VARCHAR(42) NOT NULL,
  airdrop_id NUMERIC(78,0) NOT NULL,
  name TEXT NOT NULL,
  merkle_root VARCHAR(66) NOT NULL,
  total_reward NUMERIC(78,0) NOT NULL,
  tree_version NUMERIC(78,0) NOT NULL,
  start_time BIGINT NOT NULL,
  end_time BIGINT NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT FALSE,
  claimed_reward NUMERIC(78,0) NOT NULL DEFAULT 0,
  last_synced_block BIGINT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (chain_id, contract_address, airdrop_id)
);
CREATE INDEX IF NOT EXISTS idx_airdrop_activity_active ON airdrop_activity (is_active);
CREATE INDEX IF NOT EXISTS idx_airdrop_activity_time ON airdrop_activity (start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_airdrop_activity_root ON airdrop_activity (merkle_root);

-- 用户分配表（从默克尔分发快照导入）
-- 用于支持「可领取奖励预览」和「用户可参与空投列表」。
CREATE TABLE IF NOT EXISTS airdrop_user_allocations (
  id BIGSERIAL PRIMARY KEY,
  chain_id INTEGER NOT NULL,
  contract_address VARCHAR(42) NOT NULL,
  airdrop_id NUMERIC(78,0) NOT NULL,
  user_address VARCHAR(42) NOT NULL,
  tree_version NUMERIC(78,0) NOT NULL,
  total_reward NUMERIC(78,0) NOT NULL,
  leaf_hash VARCHAR(66),
  import_batch_id VARCHAR(64),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (chain_id, contract_address, airdrop_id, user_address, tree_version)
);
CREATE INDEX IF NOT EXISTS idx_allocations_airdrop_user ON airdrop_user_allocations (airdrop_id, user_address);
CREATE INDEX IF NOT EXISTS idx_allocations_tree_version ON airdrop_user_allocations (tree_version);
CREATE INDEX IF NOT EXISTS idx_allocations_contract ON airdrop_user_allocations (contract_address);

-- 用户状态视图：聚合用户总奖励、已领取与待领取
-- 规则：按最新 tree_version 选择分配；已领取占位为 0（后续可替换为事件聚合）。
DROP VIEW IF EXISTS airdrop_user_status;
CREATE OR REPLACE VIEW airdrop_user_status AS
WITH latest_alloc AS (
  SELECT DISTINCT ON (chain_id, contract_address, airdrop_id, user_address)
         chain_id, contract_address, airdrop_id, user_address,
         tree_version, total_reward
  FROM airdrop_user_allocations
  ORDER BY chain_id, contract_address, airdrop_id, user_address, tree_version DESC
), claimed AS (
  -- Fallback：事件表 event_reward_claimed 暂未创建或不可用，
  -- 使用分配表占位，按用户最近版本的分配生成键，并将 claimed_reward 置为 0。
  SELECT DISTINCT chain_id,
         contract_address,
         airdrop_id,
         user_address,
         0::NUMERIC(78,0) AS claimed_reward
  FROM airdrop_user_allocations
)
SELECT la.chain_id,
       la.contract_address,
       la.airdrop_id,
       la.user_address,
       la.tree_version,
       la.total_reward,
       COALESCE(c.claimed_reward, 0) AS claimed_reward,
       GREATEST(la.total_reward - COALESCE(c.claimed_reward, 0), 0) AS pending_reward,
       TRUE AS has_record
FROM latest_alloc la
LEFT JOIN claimed c
  ON c.chain_id = la.chain_id AND c.contract_address = la.contract_address
 AND c.airdrop_id = la.airdrop_id AND c.user_address = la.user_address;

-- 用户可参与空投视图：筛选已激活且在时间窗内，并存在分配记录
DROP VIEW IF EXISTS airdrop_user_eligible_airdrops;
CREATE OR REPLACE VIEW airdrop_user_eligible_airdrops AS
SELECT aa.chain_id,
       aa.contract_address,
       aa.airdrop_id,
       aa.name,
       aa.start_time,
       aa.end_time,
       aa.is_active,
       au.user_address,
       au.total_reward,
       au.tree_version
FROM airdrop_activity aa
JOIN (
  -- 选取每个用户在每个空投下的最新分配（按 tree_version 降序）
  SELECT DISTINCT ON (chain_id, contract_address, airdrop_id, user_address)
         chain_id, contract_address, airdrop_id, user_address,
         tree_version, total_reward
  FROM airdrop_user_allocations
  ORDER BY chain_id, contract_address, airdrop_id, user_address, tree_version DESC
) au
  ON au.chain_id = aa.chain_id AND au.contract_address = aa.contract_address AND au.airdrop_id = aa.airdrop_id
WHERE aa.is_active = TRUE
  AND EXTRACT(EPOCH FROM NOW())::BIGINT BETWEEN aa.start_time AND aa.end_time;

COMMIT;