package service

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mumu/cryptoSwap/src/app/api/dto"
	"github.com/mumu/cryptoSwap/src/app/model"
	"github.com/mumu/cryptoSwap/src/core/ctx"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type StakeService struct{}

func NewStakeService() *StakeService {
	return &StakeService{}
}

// ProcessStake 处理质押逻辑
func (s *StakeService) ProcessStake(userAddress string, chainId int64, amount float64, token string) (*model.StakeRecord, error) {
	// 1. 验证用户地址和代币有效性
	if !common.IsHexAddress(userAddress) {
		return nil, fmt.Errorf("无效的用户地址: %s", userAddress)
	}

	// 2. 检查余额是否足够（这里需要调用代币合约检查余额）
	// 在实际实现中，这里应该调用智能合约检查用户余额
	// balance, err := s.checkTokenBalance(userAddress, token, chainId)
	// if err != nil {
	//     return nil, fmt.Errorf("检查余额失败: %v", err)
	// }
	// if balance < amount {
	//     return nil, fmt.Errorf("余额不足，当前余额: %f, 需要: %f", balance, amount)
	// }

	// 3. 调用智能合约进行质押
	// 在实际实现中，这里应该调用质押合约的质押方法
	// txHash, err := s.callStakeContract(userAddress, amount, token, chainId)
	// if err != nil {
	//     return nil, fmt.Errorf("质押合约调用失败: %v", err)
	// }

	// 4. 记录质押信息到UserOperationRecord表
	operationRecord := &model.UserOperationRecord{
		ChainId:       chainId,
		Address:       userAddress,
		Amount:        int64(amount * 1e8), // 转换为整数存储，假设8位小数
		TokenAddress:  token,
		OperationTime: time.Now(),
		UnlockTime:    time.Now().Add(7 * 24 * time.Hour),              // 示例：7天锁定期
		TxHash:        "0x" + fmt.Sprintf("%x", time.Now().UnixNano()), // 示例交易哈希
		EventType:     "stake",
	}

	if err := ctx.Ctx.DB.Create(operationRecord).Error; err != nil {
		return nil, fmt.Errorf("创建质押记录失败: %v", err)
	}

	// 5. 更新用户积分（基于ScoreRules计算）
	go s.updateUserScore(userAddress, chainId, token, amount)

	// 6. 返回StakeRecord格式的数据
	stakeRecord := &model.StakeRecord{
		ID:          fmt.Sprintf("%d", operationRecord.Id),
		UserAddress: userAddress,
		ChainId:     chainId,
		Amount:      amount,
		Token:       token,
		Status:      "active",
		CreatedAt:   operationRecord.OperationTime,
		UpdatedAt:   operationRecord.OperationTime,
	}

	return stakeRecord, nil
}

// ProcessWithdraw 处理提取逻辑
func (s *StakeService) ProcessWithdraw(userAddress string, chainId int64, stakeId string) (*model.StakeRecord, error) {
	// 1. 验证质押记录存在且属于该用户
	var operationRecord model.UserOperationRecord
	if err := ctx.Ctx.DB.Where("id = ? AND address = ? AND chain_id = ? AND event_type = ?",
		stakeId, userAddress, chainId, "stake").First(&operationRecord).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("质押记录不存在或不属于该用户")
		}
		return nil, fmt.Errorf("查询质押记录失败: %v", err)
	}

	// 2. 检查质押是否可提取（检查锁定期）
	if time.Now().Before(operationRecord.UnlockTime) {
		return nil, fmt.Errorf("质押未过锁定期，解锁时间: %s", operationRecord.UnlockTime.Format("2006-01-02 15:04:05"))
	}

	// 3. 调用智能合约进行提取
	// 在实际实现中，这里应该调用质押合约的提取方法
	// txHash, err := s.callWithdrawContract(stakeId, chainId)
	// if err != nil {
	//     return nil, fmt.Errorf("提取合约调用失败: %v", err)
	// }

	// 4. 创建提取记录
	withdrawRecord := &model.UserOperationRecord{
		ChainId:       chainId,
		Address:       userAddress,
		Amount:        operationRecord.Amount, // 提取相同金额
		TokenAddress:  operationRecord.TokenAddress,
		OperationTime: time.Now(),
		TxHash:        "0x" + fmt.Sprintf("%x", time.Now().UnixNano()), // 示例交易哈希
		EventType:     "withdraw",
	}

	if err := ctx.Ctx.DB.Create(withdrawRecord).Error; err != nil {
		return nil, fmt.Errorf("创建提取记录失败: %v", err)
	}

	// 5. 返回StakeRecord格式的数据
	stakeRecord := &model.StakeRecord{
		ID:          stakeId,
		UserAddress: userAddress,
		ChainId:     chainId,
		Amount:      float64(operationRecord.Amount) / 1e8, // 转换回浮点数
		Token:       operationRecord.TokenAddress,
		Status:      "withdrawn",
		CreatedAt:   operationRecord.OperationTime,
		UpdatedAt:   time.Now(),
	}

	return stakeRecord, nil
}

// GetStakeRecords 获取质押记录
func (s *StakeService) GetStakeRecords(userAddress string, chainId int64, pagination dto.Pagination) ([]model.StakeRecord, int64, error) {
	var operationRecords []model.UserOperationRecord
	var total int64

	// 构建查询条件（只查询质押事件）
	query := ctx.Ctx.DB.Where("address = ? AND event_type = ?", userAddress, "stake")
	if chainId > 0 {
		query = query.Where("chain_id = ?", chainId)
	}

	// 获取总数
	if err := query.Model(&model.UserOperationRecord{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询记录总数失败: %v", err)
	}

	// 应用分页查询记录
	if err := query.Order("operation_time DESC").
		Offset(pagination.Offset).
		Limit(pagination.PageSize).
		Find(&operationRecords).Error; err != nil {
		return nil, 0, fmt.Errorf("查询质押记录失败: %v", err)
	}

	// 转换为StakeRecord格式
	var stakeRecords []model.StakeRecord
	for _, record := range operationRecords {
		// 检查是否已提取
		status := "active"
		var withdrawRecord model.UserOperationRecord
		if err := ctx.Ctx.DB.Where("address = ? AND chain_id = ? AND event_type = ? AND amount = ?",
			userAddress, chainId, "withdraw", record.Amount).First(&withdrawRecord).Error; err == nil {
			status = "withdrawn"
		}

		stakeRecord := model.StakeRecord{
			ID:          fmt.Sprintf("%d", record.Id),
			UserAddress: record.Address,
			ChainId:     record.ChainId,
			Amount:      float64(record.Amount) / 1e8,
			Token:       record.TokenAddress,
			Status:      status,
			CreatedAt:   record.OperationTime,
			UpdatedAt:   record.OperationTime,
		}
		stakeRecords = append(stakeRecords, stakeRecord)
	}

	return stakeRecords, total, nil
}

// GetStakeOverview 获取质押概览
func (s *StakeService) GetStakeOverview(userAddress string, chainId int64) (*model.StakeOverview, error) {
	var totalStaked int64
	var activeStakes int64

	// 构建查询条件
	query := ctx.Ctx.DB.Model(&model.UserOperationRecord{}).Where("address = ? AND event_type = ?", userAddress, "stake")
	if chainId > 0 {
		query = query.Where("chain_id = ?", chainId)
	}

	// 计算总质押量（活跃状态的质押）
	if err := query.Select("COALESCE(SUM(amount), 0)").Scan(&totalStaked).Error; err != nil {
		return nil, fmt.Errorf("计算总质押量失败: %v", err)
	}

	// 计算活跃质押数量（未提取的质押记录）
	activeQuery := query
	if chainId > 0 {
		activeQuery = activeQuery.Where("chain_id = ?", chainId)
	}

	// 统计未提取的质押记录数量
	var stakeRecords []model.UserOperationRecord
	if err := activeQuery.Find(&stakeRecords).Error; err != nil {
		return nil, fmt.Errorf("查询质押记录失败: %v", err)
	}

	for _, record := range stakeRecords {
		var withdrawRecord model.UserOperationRecord
		if err := ctx.Ctx.DB.Where("address = ? AND chain_id = ? AND event_type = ? AND amount = ?",
			userAddress, record.ChainId, "withdraw", record.Amount).First(&withdrawRecord).Error; err != nil {
			// 如果没有找到对应的提取记录，说明是活跃质押
			activeStakes++
		}
	}

	// 计算总收益（从Users表获取积分信息）
	totalRewards, err := s.getUserRewards(userAddress, chainId)
	if err != nil {
		return nil, fmt.Errorf("获取用户收益失败: %v", err)
	}

	overview := &model.StakeOverview{
		TotalStaked:  float64(totalStaked) / 1e8,
		TotalRewards: totalRewards,
		ActiveStakes: int(activeStakes),
		UserAddress:  userAddress,
		ChainId:      chainId,
	}

	return overview, nil
}

// updateUserScore 更新用户积分
func (s *StakeService) updateUserScore(userAddress string, chainId int64, token string, amount float64) {
	// 1. 获取积分规则
	var scoreRule model.ScoreRules
	if err := ctx.Ctx.DB.Where("chain_id = ? AND token_address = ?", chainId, token).First(&scoreRule).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果没有找到积分规则，使用默认规则
			scoreRule.Score = decimal.NewFromFloat(1.0) // 默认1:1积分
			scoreRule.Decimals = 8
		} else {
			fmt.Printf("查询积分规则失败: %v\n", err)
			return
		}
	}

	// 2. 计算应得积分
	amountDecimal := decimal.NewFromFloat(amount)
	score := amountDecimal.Mul(scoreRule.Score)

	// 3. 更新用户积分信息
	var user model.Users
	if err := ctx.Ctx.DB.Where("chain_id = ? AND address = ?", chainId, userAddress).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建新用户记录
			user = model.Users{
				ChainId:      chainId,
				Address:      userAddress,
				TokenAddress: token,
				TotalAmount:  int64(amount * 1e8),
				JfAmount:     score.IntPart(),
				Jf:           score,
				JfTime:       time.Now(),
			}
			if err := ctx.Ctx.DB.Create(&user).Error; err != nil {
				fmt.Printf("创建用户记录失败: %v\n", err)
				return
			}
		} else {
			fmt.Printf("查询用户记录失败: %v\n", err)
			return
		}
	} else {
		// 更新现有用户记录
		user.TotalAmount += int64(amount * 1e8)
		user.JfAmount += score.IntPart()
		user.Jf = user.Jf.Add(score)
		user.JfTime = time.Now()

		if err := ctx.Ctx.DB.Save(&user).Error; err != nil {
			fmt.Printf("更新用户积分失败: %v\n", err)
			return
		}
	}

	fmt.Printf("用户 %s 积分更新成功，当前积分: %s\n", userAddress, user.Jf.String())
}

// getUserRewards 获取用户收益（从积分信息计算）
func (s *StakeService) getUserRewards(userAddress string, chainId int64) (float64, error) {
	var user model.Users
	if err := ctx.Ctx.DB.Where("chain_id = ? AND address = ?", chainId, userAddress).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil // 用户不存在，返回0收益
		}
		return 0, fmt.Errorf("查询用户信息失败: %v", err)
	}

	// 将积分转换为收益（这里需要根据实际业务规则）
	// 示例：1积分 = 0.01代币
	rewardRate := decimal.NewFromFloat(0.01)
	totalRewards := user.Jf.Mul(rewardRate)

	reward, _ := totalRewards.Float64()
	return reward, nil
}

// 以下为辅助方法（在实际实现中需要完成）

// checkTokenBalance 检查代币余额
func (s *StakeService) checkTokenBalance(userAddress, token string, chainId int64) (float64, error) {
	// 实现代币余额检查逻辑
	// 需要调用代币合约的balanceOf方法
	return 1000.0, nil // 示例返回值
}

// callStakeContract 调用质押合约
func (s *StakeService) callStakeContract(userAddress string, amount float64, token string, chainId int64) (string, error) {
	// 实现质押合约调用逻辑
	// 返回交易哈希
	return "0x" + fmt.Sprintf("%x", time.Now().UnixNano()), nil // 示例交易哈希
}

// callWithdrawContract 调用提取合约
func (s *StakeService) callWithdrawContract(stakeId string, chainId int64) (string, error) {
	// 实现提取合约调用逻辑
	// 返回交易哈希
	return "0x" + fmt.Sprintf("%x", time.Now().UnixNano()), nil // 示例交易哈希
}
