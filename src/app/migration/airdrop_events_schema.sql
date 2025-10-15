-- Schema for MerkleAirdrop events: RewardClaimed & UpdateTotalRewardUpdated
-- 地址与交易哈希统一建议在应用层转换为小写后再入库
-- 金额统一以 wei 存储（numeric(78,0)），避免精度误差
-- 使用唯一键 (tx_hash, log_index) 保证事件入库幂等等

BEGIN;

-- 事件表：RewardClaimed
CREATE TABLE IF NOT EXISTS reward_claimed_events (
  id BIGSERIAL PRIMARY KEY,
  chain_id INTEGER NOT NULL,
  contract_address TEXT NOT NULL CHECK (contract_address ~ '^0x[0-9a-f]{40}$'),
  airdrop_id NUMERIC(78, 0) NOT NULL,
  user_address TEXT NOT NULL CHECK (user_address ~ '^0x[0-9a-f]{40}$'),
  claim_amount NUMERIC(78, 0) NOT NULL,
  total_reward NUMERIC(78, 0) NOT NULL,
  claimed_reward NUMERIC(78, 0) NOT NULL,
  pending_reward NUMERIC(78, 0) NOT NULL,
  event_timestamp TIMESTAMPTZ NOT NULL,
  block_number BIGINT NOT NULL,
  tx_hash TEXT NOT NULL CHECK (tx_hash ~ '^0x[0-9a-f]{64}$'),
  log_index INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (claim_amount >= 0 AND total_reward >= 0 AND claimed_reward >= 0 AND pending_reward >= 0),
  UNIQUE (tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_reward_claimed_airdrop_user
  ON reward_claimed_events (airdrop_id, user_address);

CREATE INDEX IF NOT EXISTS idx_reward_claimed_contract_block
  ON reward_claimed_events (contract_address, block_number DESC);

-- 事件表：UpdateTotalRewardUpdated
CREATE TABLE IF NOT EXISTS total_reward_updates (
  id BIGSERIAL PRIMARY KEY,
  chain_id INTEGER NOT NULL,
  contract_address TEXT NOT NULL CHECK (contract_address ~ '^0x[0-9a-f]{40}$'),
  airdrop_id NUMERIC(78, 0) NOT NULL,
  user_address TEXT NOT NULL CHECK (user_address ~ '^0x[0-9a-f]{40}$'),
  total_reward NUMERIC(78, 0) NOT NULL,
  claimed_reward NUMERIC(78, 0) NOT NULL,
  pending_reward NUMERIC(78, 0) NOT NULL,
  event_timestamp TIMESTAMPTZ NOT NULL,
  block_number BIGINT NOT NULL,
  tx_hash TEXT NOT NULL CHECK (tx_hash ~ '^0x[0-9a-f]{64}$'),
  log_index INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (total_reward >= 0 AND claimed_reward >= 0 AND pending_reward >= 0),
  UNIQUE (tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_total_updates_airdrop_user
  ON total_reward_updates (airdrop_id, user_address);

CREATE INDEX IF NOT EXISTS idx_total_updates_contract_block
  ON total_reward_updates (contract_address, block_number DESC);

-- 聚合状态表：每个 (airdrop_id, user_address) 的最新快照
CREATE TABLE IF NOT EXISTS user_reward_state (
  airdrop_id NUMERIC(78,0) NOT NULL,
  user_address TEXT NOT NULL CHECK (user_address ~ '^0x[0-9a-f]{40}$'),
  total_reward NUMERIC(78,0) NOT NULL DEFAULT 0,
  claimed_reward NUMERIC(78,0) NOT NULL DEFAULT 0,
  pending_reward NUMERIC(78,0) NOT NULL DEFAULT 0,
  last_event_block_number BIGINT,
  last_event_tx_hash TEXT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (total_reward >= 0 AND claimed_reward >= 0 AND pending_reward >= 0),
  CHECK (last_event_tx_hash IS NULL OR last_event_tx_hash ~ '^0x[0-9a-f]{64}$'),
  PRIMARY KEY (airdrop_id, user_address)
);

CREATE INDEX IF NOT EXISTS idx_user_reward_state_pending
  ON user_reward_state (airdrop_id, pending_reward DESC);

COMMIT;

-- 使用说明：
-- 1) 地址与 tx_hash 推荐在应用层先转为小写再入库，以满足 CHECK。
-- 2) 事件中的 timestamp（uint256 秒）建议在插入时用 TO_TIMESTAMP(...) 转为 TIMESTAMPTZ。
-- 3) 事件入库以 (tx_hash, log_index) 做幂等等约束，防止重复消费。
-- 4) 更新 user_reward_state 时以更大的 block_number 覆盖，处理链上重组。