-- AlanSwap 其他表结构迁移文件
-- 此文件包含除流动性池相关表之外的其他业务表

-- 链信息表
CREATE TABLE IF NOT EXISTS chain (
    id BIGSERIAL PRIMARY KEY,
    chain_id BIGINT NOT NULL,
    chain_name VARCHAR(100) NOT NULL,
    address VARCHAR(42) NOT NULL,
    last_block_num BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(chain_id)
);

-- 积分规则表
CREATE TABLE IF NOT EXISTS score_rules (
    id BIGSERIAL PRIMARY KEY,
    chain_id BIGINT NOT NULL,
    token_address VARCHAR(42) NOT NULL,
    score DECIMAL(30,18) NOT NULL DEFAULT '0',
    decimals BIGINT NOT NULL DEFAULT 18,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(chain_id, token_address)
);

-- 用户操作记录表
CREATE TABLE IF NOT EXISTS user_operation_record (
    id BIGSERIAL PRIMARY KEY,
    chain_id BIGINT NOT NULL,
    address VARCHAR(42) NOT NULL,
    pool_id BIGINT NOT NULL,
    amount BIGINT NOT NULL DEFAULT 0,
    operation_time TIMESTAMP NOT NULL,
    unlock_time TIMESTAMP NOT NULL,
    tx_hash VARCHAR(66) NOT NULL,
    block_number BIGINT NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    token_address VARCHAR(42) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    chain_id BIGINT NOT NULL,
    address VARCHAR(42) NOT NULL,
    token_address VARCHAR(42) NOT NULL,
    total_amount BIGINT NOT NULL DEFAULT 0,
    last_block_num BIGINT NOT NULL DEFAULT 0,
    jf_amount BIGINT NOT NULL DEFAULT 0,
    jf_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    jf DECIMAL(30,18) NOT NULL DEFAULT '0',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(chain_id, address, token_address)
);

-- 创建索引
-- chain表索引
CREATE INDEX IF NOT EXISTS idx_chain_chain_id ON chain(chain_id);
CREATE INDEX IF NOT EXISTS idx_chain_last_block_num ON chain(last_block_num);

-- score_rules表索引
CREATE INDEX IF NOT EXISTS idx_score_rules_chain_id ON score_rules(chain_id);
CREATE INDEX IF NOT EXISTS idx_score_rules_token_address ON score_rules(token_address);

-- user_operation_record表索引
CREATE INDEX IF NOT EXISTS idx_user_operation_record_chain_id ON user_operation_record(chain_id);
CREATE INDEX IF NOT EXISTS idx_user_operation_record_address ON user_operation_record(address);
CREATE INDEX IF NOT EXISTS idx_user_operation_record_pool_id ON user_operation_record(pool_id);
CREATE INDEX IF NOT EXISTS idx_user_operation_record_tx_hash ON user_operation_record(tx_hash);
CREATE INDEX IF NOT EXISTS idx_user_operation_record_block_number ON user_operation_record(block_number);
CREATE INDEX IF NOT EXISTS idx_user_operation_record_event_type ON user_operation_record(event_type);
CREATE INDEX IF NOT EXISTS idx_user_operation_record_token_address ON user_operation_record(token_address);
CREATE INDEX IF NOT EXISTS idx_user_operation_record_operation_time ON user_operation_record(operation_time);

-- users表索引
CREATE INDEX IF NOT EXISTS idx_users_chain_id ON users(chain_id);
CREATE INDEX IF NOT EXISTS idx_users_address ON users(address);
CREATE INDEX IF NOT EXISTS idx_users_token_address ON users(token_address);
CREATE INDEX IF NOT EXISTS idx_users_last_block_num ON users(last_block_num);
CREATE INDEX IF NOT EXISTS idx_users_jf_time ON users(jf_time);

-- 添加表注释
COMMENT ON TABLE chain IS '区块链信息表';
COMMENT ON TABLE score_rules IS '积分规则表';
COMMENT ON TABLE user_operation_record IS '用户操作记录表';
COMMENT ON TABLE users IS '用户信息表';

-- 添加字段注释
-- chain表字段注释
COMMENT ON COLUMN chain.id IS '主键ID';
COMMENT ON COLUMN chain.chain_id IS '链ID';
COMMENT ON COLUMN chain.chain_name IS '链名称';
COMMENT ON COLUMN chain.address IS '合约地址';
COMMENT ON COLUMN chain.last_block_num IS '最后处理的区块号';

-- score_rules表字段注释
COMMENT ON COLUMN score_rules.id IS '主键ID';
COMMENT ON COLUMN score_rules.chain_id IS '链ID';
COMMENT ON COLUMN score_rules.token_address IS '代币合约地址';
COMMENT ON COLUMN score_rules.score IS '积分值';
COMMENT ON COLUMN score_rules.decimals IS '代币精度';

-- user_operation_record表字段注释
COMMENT ON COLUMN user_operation_record.id IS '主键ID';
COMMENT ON COLUMN user_operation_record.chain_id IS '链ID';
COMMENT ON COLUMN user_operation_record.address IS '用户地址';
COMMENT ON COLUMN user_operation_record.pool_id IS '流动性池ID';
COMMENT ON COLUMN user_operation_record.amount IS '操作数量';
COMMENT ON COLUMN user_operation_record.operation_time IS '操作时间';
COMMENT ON COLUMN user_operation_record.unlock_time IS '解锁时间';
COMMENT ON COLUMN user_operation_record.tx_hash IS '交易哈希';
COMMENT ON COLUMN user_operation_record.block_number IS '区块号';
COMMENT ON COLUMN user_operation_record.event_type IS '事件类型';
COMMENT ON COLUMN user_operation_record.token_address IS '代币地址';

-- users表字段注释
COMMENT ON COLUMN users.id IS '主键ID';
COMMENT ON COLUMN users.chain_id IS '链ID';
COMMENT ON COLUMN users.address IS '用户地址';
COMMENT ON COLUMN users.token_address IS '代币地址';
COMMENT ON COLUMN users.total_amount IS '总数量';
COMMENT ON COLUMN users.last_block_num IS '最后处理的区块号';
COMMENT ON COLUMN users.jf_amount IS '积分数量';
COMMENT ON COLUMN users.jf_time IS '积分时间';
COMMENT ON COLUMN users.jf IS '积分值';