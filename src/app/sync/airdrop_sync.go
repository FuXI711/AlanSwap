package sync

import (
    "math/big"
    "time"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/mumu/cryptoSwap/src/app/model"
    "github.com/mumu/cryptoSwap/src/core/ctx"
    "github.com/mumu/cryptoSwap/src/core/log"
    "go.uber.org/zap"
    "gorm.io/gorm"
    "gorm.io/gorm/clause"
)

// parseRewardClaimedEvent 解析 RewardClaimed(uint256,address,uint256,uint256,uint256,uint256,uint256)
func parseRewardClaimedEvent(vLog types.Log, chainId int) *model.RewardClaimedEvent {
    if len(vLog.Topics) < 3 || len(vLog.Data) < 160 {
        return nil
    }

    // indexed: airdropId, user
    airdropId := new(big.Int).SetBytes(common.TrimLeftZeroes(vLog.Topics[1].Bytes()))
    user := common.BytesToAddress(vLog.Topics[2].Bytes()).Hex()

    data := vLog.Data
    claimAmount := new(big.Int).SetBytes(common.TrimLeftZeroes(data[0:32]))
    totalReward := new(big.Int).SetBytes(common.TrimLeftZeroes(data[32:64]))
    claimedReward := new(big.Int).SetBytes(common.TrimLeftZeroes(data[64:96]))
    pendingReward := new(big.Int).SetBytes(common.TrimLeftZeroes(data[96:128]))
    ts := new(big.Int).SetBytes(common.TrimLeftZeroes(data[128:160]))
    eventTime := time.Unix(ts.Int64(), 0)

    return &model.RewardClaimedEvent{
        ChainId:         int64(chainId),
        ContractAddress: vLog.Address.Hex(),
        AirdropId:       airdropId.String(),
        UserAddress:     user,
        ClaimAmount:     claimAmount.String(),
        TotalReward:     totalReward.String(),
        ClaimedReward:   claimedReward.String(),
        PendingReward:   pendingReward.String(),
        EventTimestamp:  eventTime,
        BlockNumber:     int64(vLog.BlockNumber),
        TxHash:          vLog.TxHash.Hex(),
        LogIndex:        int(vLog.Index),
    }
}

// parseTotalRewardUpdatedEvent 解析 UpdateTotalRewardUpdated(uint256,address,uint256,uint256,uint256,uint256)
func parseTotalRewardUpdatedEvent(vLog types.Log, chainId int) *model.TotalRewardUpdatedEvent {
    if len(vLog.Topics) < 3 || len(vLog.Data) < 128 {
        return nil
    }

    airdropId := new(big.Int).SetBytes(common.TrimLeftZeroes(vLog.Topics[1].Bytes()))
    user := common.BytesToAddress(vLog.Topics[2].Bytes()).Hex()

    data := vLog.Data
    totalReward := new(big.Int).SetBytes(common.TrimLeftZeroes(data[0:32]))
    claimedReward := new(big.Int).SetBytes(common.TrimLeftZeroes(data[32:64]))
    pendingReward := new(big.Int).SetBytes(common.TrimLeftZeroes(data[64:96]))
    ts := new(big.Int).SetBytes(common.TrimLeftZeroes(data[96:128]))
    eventTime := time.Unix(ts.Int64(), 0)

    return &model.TotalRewardUpdatedEvent{
        ChainId:         int64(chainId),
        ContractAddress: vLog.Address.Hex(),
        AirdropId:       airdropId.String(),
        UserAddress:     user,
        TotalReward:     totalReward.String(),
        ClaimedReward:   claimedReward.String(),
        PendingReward:   pendingReward.String(),
        EventTimestamp:  eventTime,
        BlockNumber:     int64(vLog.BlockNumber),
        TxHash:          vLog.TxHash.Hex(),
        LogIndex:        int(vLog.Index),
    }
}

// saveAirdropEvents 批量保存空投事件，并更新区块高度
func saveAirdropEvents(rewardClaimed []*model.RewardClaimedEvent, totalUpdates []*model.TotalRewardUpdatedEvent, chainId int, targetBlockNum uint64) error {
    return ctx.Ctx.DB.Transaction(func(tx *gorm.DB) error {
        // RewardClaimedEvents 去重插入
        if len(rewardClaimed) > 0 {
            if err := tx.Clauses(clause.OnConflict{
                Columns:   []clause.Column{{Name: "tx_hash"}, {Name: "log_index"}},
                DoNothing: true,
            }).CreateInBatches(rewardClaimed, 100).Error; err != nil {
                log.Logger.Error("批量插入 RewardClaimed 事件失败", zap.Error(err))
                return err
            }
            // 更新用户快照（基于最新事件）
            for _, e := range rewardClaimed {
                if err := upsertUserRewardState(tx, e.AirdropId, e.UserAddress, e.TotalReward, e.ClaimedReward, e.PendingReward, e.BlockNumber, e.TxHash); err != nil {
                    log.Logger.Error("更新用户快照失败", zap.Error(err))
                    return err
                }
            }
        }

        // TotalRewardUpdatedEvents 去重插入
        if len(totalUpdates) > 0 {
            if err := tx.Clauses(clause.OnConflict{
                Columns:   []clause.Column{{Name: "tx_hash"}, {Name: "log_index"}},
                DoNothing: true,
            }).CreateInBatches(totalUpdates, 100).Error; err != nil {
                log.Logger.Error("批量插入 TotalRewardUpdated 事件失败", zap.Error(err))
                return err
            }
            for _, e := range totalUpdates {
                if err := upsertUserRewardState(tx, e.AirdropId, e.UserAddress, e.TotalReward, e.ClaimedReward, e.PendingReward, e.BlockNumber, e.TxHash); err != nil {
                    log.Logger.Error("更新用户快照失败", zap.Error(err))
                    return err
                }
            }
        }

        // 更新链区块高度
        return updateBlockNumber(chainId, targetBlockNum)
    })
}

// upsertUserRewardState 写入/更新用户快照，按更大的区块覆盖
func upsertUserRewardState(tx *gorm.DB, airdropId, userAddr, total, claimed, pending string, blockNumber int64, txHash string) error {
    // 使用原生 SQL，确保只在更大的 block_number 时更新
    return tx.Exec(`
        INSERT INTO user_reward_state (airdrop_id, user_address, total_reward, claimed_reward, pending_reward, last_event_block_number, last_event_tx_hash, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, NOW())
        ON CONFLICT (airdrop_id, user_address) DO UPDATE
        SET total_reward = EXCLUDED.total_reward,
            claimed_reward = EXCLUDED.claimed_reward,
            pending_reward = EXCLUDED.pending_reward,
            last_event_block_number = EXCLUDED.last_event_block_number,
            last_event_tx_hash = EXCLUDED.last_event_tx_hash,
            updated_at = NOW()
        WHERE user_reward_state.last_event_block_number IS NULL OR EXCLUDED.last_event_block_number >= user_reward_state.last_event_block_number
    `, airdropId, userAddr, total, claimed, pending, blockNumber, txHash).Error
}