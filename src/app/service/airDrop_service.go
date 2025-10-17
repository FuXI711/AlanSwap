package service

import (
    "database/sql"
    "strings"
    "time"

    "github.com/shopspring/decimal"
    "github.com/mumu/cryptoSwap/src/core/ctx"
)

type AirDropService struct{}

func NewAirDropService() *AirDropService { return &AirDropService{} }

// OverviewDTO 返回概览数据
type OverviewDTO struct {
    TotalRewards              string
    TotalRewardsWeeklyChange  string
    ClaimedRewards            string
    ClaimedRewardsValue       string
    PendingRewards            string
    PendingRewardsValue       string
    TokenSymbol               string
}

// AvailableItemDTO 可参与空投的条目
type AvailableItemDTO struct {
    AirdropId         string
    Name              string
    Description       string
    AirDropIcon       string
    TokenSymbol       string
    TotalReward       string
    UserTotalReward   string
    UserClaimedReward string
    UserPendingReward string
    StartTime         time.Time
    EndTime           time.Time
    UserCount         int64
    Status            string
    StatusDesc        string
    TaskList          []TaskItemDTO
}

type TaskItemDTO struct {
    TaskId   int64
    TaskName string
    Status   int // 0未完成 1已完成
}

// RankingItemDTO 排行榜条目
type RankingItemDTO struct {
    Rank                 int64
    WalletAddress        string
    ClaimAmount          string
    ClaimAmountFormatted string
    LastClaimTime        int64
    LastClaimTimeFormatted string
}

type RankingDTO struct {
    TotalUsers     int64
    CurrentUserRank int64
    List           []RankingItemDTO
    TokenSymbol    string
    UpdateTime     int64
}

// Overview 根据钱包地址返回奖励概览
func (s *AirDropService) Overview(walletAddress string) (*OverviewDTO, error) {
    addr := strings.ToLower(walletAddress)

    // tokenSymbol 取首个活动的符号（默认 CSWAP）
    var tokenSymbol string = "CSWAP"
    _ = ctx.Ctx.DB.Raw("SELECT token_symbol FROM airdrop_campaigns WHERE is_active = TRUE ORDER BY updated_at DESC LIMIT 1").Scan(&tokenSymbol)
    if tokenSymbol == "" {
        tokenSymbol = "CSWAP"
    }

    // 总奖励来自 whitelist（若无数据则为 0）
    var totalRewardsStr sql.NullString
    err := ctx.Ctx.DB.Raw("SELECT COALESCE(SUM(total_reward)::text, '0') FROM airdrop_whitelist WHERE wallet_address = ?", addr).Scan(&totalRewardsStr).Error
    if err != nil { return nil, err }
    totalRewards := decimal.RequireFromString(nilToZero(totalRewardsStr))

    // 已领取奖励
    var claimedStr sql.NullString
    if err := ctx.Ctx.DB.Raw("SELECT COALESCE(SUM(claim_amount)::text, '0') FROM reward_claimed_events WHERE user_address = ?", addr).Scan(&claimedStr).Error; err != nil {
        return nil, err
    }
    claimed := decimal.RequireFromString(nilToZero(claimedStr))

    // 待领取奖励
    pending := totalRewards.Sub(claimed)
    if pending.IsNegative() { pending = decimal.Zero }

    // 最近 7 天的总奖励变化（用已领取近 7 天作为近似）
    var weeklyChangeStr sql.NullString
    if err := ctx.Ctx.DB.Raw("SELECT COALESCE(SUM(claim_amount)::text, '0') FROM reward_claimed_events WHERE user_address = ? AND event_timestamp >= NOW() - INTERVAL '7 days'", addr).Scan(&weeklyChangeStr).Error; err != nil {
        return nil, err
    }
    weekly := decimal.RequireFromString(nilToZero(weeklyChangeStr))

    dto := &OverviewDTO{
        TotalRewards:             totalRewards.String() + " " + tokenSymbol,
        TotalRewardsWeeklyChange: "+" + weekly.String() + " " + tokenSymbol,
        ClaimedRewards:           claimed.String() + " " + tokenSymbol,
        ClaimedRewardsValue:      claimed.String(),
        PendingRewards:           pending.String() + " " + tokenSymbol,
        PendingRewardsValue:      pending.String(),
        TokenSymbol:              tokenSymbol,
    }
    return dto, nil
}

// Available 返回可参与空投活动列表（分页）
func (s *AirDropService) Available(walletAddress string, page, size int) (int64, []AvailableItemDTO, error) {
    addr := strings.ToLower(walletAddress)
    if page <= 0 { page = 1 }
    if size <= 0 { size = 20 }
    offset := (page - 1) * size

    // 查询总数
    var total int64
    if err := ctx.Ctx.DB.Raw("SELECT COUNT(*) FROM airdrop_campaigns").Scan(&total).Error; err != nil {
        return 0, nil, err
    }

    // 查询活动基本信息
    rows, err := ctx.Ctx.DB.Raw("SELECT airdrop_id::text, name, description, icon_url, token_symbol, total_reward::text, start_time, end_time, is_active FROM airdrop_campaigns ORDER BY updated_at DESC OFFSET ? LIMIT ?", offset, size).Rows()
    if err != nil { return 0, nil, err }
    defer rows.Close()

    list := make([]AvailableItemDTO, 0)
    for rows.Next() {
        var (
            airdropId   string
            name        string
            description sql.NullString
            iconUrl     sql.NullString
            tokenSymbol string
            totalReward string
            startTime   sql.NullTime
            endTime     sql.NullTime
            isActive    bool
        )
        if err := rows.Scan(&airdropId, &name, &description, &iconUrl, &tokenSymbol, &totalReward, &startTime, &endTime, &isActive); err != nil {
            return 0, nil, err
        }
        var item AvailableItemDTO
        item.AirdropId = airdropId
        item.Name = name
        item.Description = nilToEmpty(description)
        item.AirDropIcon = nilToEmpty(iconUrl)
        item.TokenSymbol = tokenSymbol
        item.TotalReward = totalReward
        if startTime.Valid { item.StartTime = startTime.Time } else { item.StartTime = time.Time{} }
        if endTime.Valid { item.EndTime = endTime.Time } else { item.EndTime = time.Time{} }

        // 用户总奖励（来自 whitelist）
        var userTotal sql.NullString
        _ = ctx.Ctx.DB.Raw("SELECT COALESCE(total_reward::text,'0') FROM airdrop_whitelist WHERE airdrop_id = ? AND wallet_address = ?", item.AirdropId, addr).Scan(&userTotal)
        item.UserTotalReward = nilToZero(userTotal)

        // 用户已领取奖励
        var userClaimed sql.NullString
        _ = ctx.Ctx.DB.Raw("SELECT COALESCE(SUM(claim_amount)::text,'0') FROM reward_claimed_events WHERE airdrop_id = ? AND user_address = ?", item.AirdropId, addr).Scan(&userClaimed)
        item.UserClaimedReward = nilToZero(userClaimed)

        // 待领取
        totalDec := decimal.RequireFromString(item.UserTotalReward)
        claimedDec := decimal.RequireFromString(item.UserClaimedReward)
        pendingDec := totalDec.Sub(claimedDec)
        if pendingDec.IsNegative() { pendingDec = decimal.Zero }
        item.UserPendingReward = pendingDec.String()

        // 用户数
        _ = ctx.Ctx.DB.Raw("SELECT COUNT(DISTINCT user_address) FROM reward_claimed_events WHERE airdrop_id = ?", item.AirdropId).Scan(&item.UserCount)

        // 状态
        now := time.Now()
        if isActive && (item.EndTime.IsZero() || now.Before(item.EndTime)) && (item.StartTime.IsZero() || now.After(item.StartTime)) {
            item.Status = "claimable"
            item.StatusDesc = "Available for collection"
        } else if !item.EndTime.IsZero() && now.After(item.EndTime) {
            item.Status = "expired"
            item.StatusDesc = "Expired"
        } else {
            item.Status = "closed"
            item.StatusDesc = "Closed"
        }

        // 任务列表（若无绑定则为空）
        item.TaskList = s.listTasksForAirdrop(addr, item.AirdropId)

        list = append(list, item)
    }

    return total, list, nil
}

func (s *AirDropService) listTasksForAirdrop(walletAddress string, airdropId string) []TaskItemDTO {
    addr := strings.ToLower(walletAddress)
    rows, err := ctx.Ctx.DB.Raw(`
        SELECT t.task_id, t.task_name,
               COALESCE(uts.user_status, 0) AS status
        FROM airdrop_task_bindings b
        JOIN tasks t ON t.task_id = b.task_id
        LEFT JOIN user_task_status uts ON uts.task_id = t.task_id AND uts.wallet_address = ?
        WHERE b.airdrop_id = ?
        ORDER BY t.task_id ASC
    `, addr, airdropId).Rows()
    if err != nil { return []TaskItemDTO{} }
    defer rows.Close()
    var res []TaskItemDTO
    for rows.Next() {
        var it TaskItemDTO
        if err := rows.Scan(&it.TaskId, &it.TaskName, &it.Status); err == nil {
            res = append(res, it)
        }
    }
    return res
}

// Ranking 排行榜
func (s *AirDropService) Ranking(airdropId string, sortBy string, page, size int, showAddress bool, currentUser string) (*RankingDTO, error) {
    if page <= 0 { page = 1 }
    if size <= 0 { size = 20 }
    offset := (page - 1) * size

    // tokenSymbol
    var tokenSymbol string = "CSWAP"
    if airdropId != "" {
        _ = ctx.Ctx.DB.Raw("SELECT token_symbol FROM airdrop_campaigns WHERE airdrop_id = ?", airdropId).Scan(&tokenSymbol)
    }
    if tokenSymbol == "" { tokenSymbol = "CSWAP" }

    // total users
    var totalUsers int64
    var where string
    if airdropId != "" { where = "WHERE airdrop_id = ?" } else { where = "" }
    if err := ctx.Ctx.DB.Raw("SELECT COUNT(DISTINCT user_address) FROM reward_claimed_events "+where, airdropId).Scan(&totalUsers).Error; err != nil {
        return nil, err
    }

    // order by
    order := "total_amount DESC"
    if sortBy == "time" { order = "last_time ASC" }

    // list（安全构造WHERE，限制排序字段为白名单）
    base := `
      SELECT user_address,
             SUM(claim_amount)::text AS total_amount,
             EXTRACT(EPOCH FROM MAX(event_timestamp))::bigint AS last_unix,
             MAX(event_timestamp) AS last_time
      FROM reward_claimed_events
    `
    groupOrder := " GROUP BY user_address ORDER BY " + order + " OFFSET ? LIMIT ?"
    var rows *sql.Rows
    var err error
    if airdropId != "" {
        rows, err = ctx.Ctx.DB.Raw(base+" WHERE airdrop_id = ?"+groupOrder, airdropId, offset, size).Rows()
    } else {
        rows, err = ctx.Ctx.DB.Raw(base+groupOrder, offset, size).Rows()
    }
    if err != nil { return nil, err }
    defer rows.Close()

    list := make([]RankingItemDTO, 0)
    var rankBase int64 = int64(offset)
    for rows.Next() {
        var addr string
        var totalAmt string
        var lastUnix int64
        var lastTime time.Time
        if err := rows.Scan(&addr, &totalAmt, &lastUnix, &lastTime); err != nil { return nil, err }
        rankBase++
        masked := ""
        if showAddress {
            if len(addr) >= 9 { // 0x + 前3 + ... + 后4 至少9位
                // 前3后4遮蔽（含0x前缀，因此取前5位：0x + 3位）
                masked = addr[:5] + "..." + addr[len(addr)-4:]
            } else {
                masked = addr
            }
        }
        list = append(list, RankingItemDTO{
            Rank:                 rankBase,
            WalletAddress:        masked,
            ClaimAmount:          totalAmt,
            ClaimAmountFormatted: totalAmt + " " + tokenSymbol,
            LastClaimTime:        lastUnix,
            LastClaimTimeFormatted: lastTime.Format("2006-01-02 15:04:05"),
        })
    }

    // current user rank
    var currentRank int64 = -1
    cu := strings.ToLower(currentUser)
    if cu != "" {
        var myTotal sql.NullString
        _ = ctx.Ctx.DB.Raw("SELECT COALESCE(SUM(claim_amount)::text,'0') FROM reward_claimed_events "+where, airdropId).Scan(&myTotal)
        myDec := decimal.RequireFromString(nilToZero(myTotal))
        var ahead int64
        // 计算排名：统计比当前用户总额大的用户数
        if airdropId != "" {
            _ = ctx.Ctx.DB.Raw(`
              SELECT COUNT(*) FROM (
                SELECT user_address, SUM(claim_amount) AS total_amount
                FROM reward_claimed_events WHERE airdrop_id = ?
                GROUP BY user_address
              ) t WHERE t.total_amount > ?
            `, airdropId, myDec.String()).Scan(&ahead)
        } else {
            _ = ctx.Ctx.DB.Raw(`
              SELECT COUNT(*) FROM (
                SELECT user_address, SUM(claim_amount) AS total_amount
                FROM reward_claimed_events
                GROUP BY user_address
              ) t WHERE t.total_amount > ?
            `, myDec.String()).Scan(&ahead)
        }
        if myDec.IsZero() {
            currentRank = -1
        } else {
            currentRank = ahead + 1
        }
    }

    dto := &RankingDTO{ TotalUsers: totalUsers, CurrentUserRank: currentRank, List: list, TokenSymbol: tokenSymbol, UpdateTime: time.Now().Unix() }
    return dto, nil
}

// ClaimProof 返回领取证明（若无则为空）
func (s *AirDropService) ClaimProof(airdropId string, walletAddress string) (string, error) {
    addr := strings.ToLower(walletAddress)
    var proof sql.NullString
    err := ctx.Ctx.DB.Raw("SELECT COALESCE(proof::text,'') FROM airdrop_whitelist WHERE airdrop_id = ? AND wallet_address = ?", airdropId, addr).Scan(&proof).Error
    if err != nil { return "", err }
    return nilToEmpty(proof), nil
}

// UserTasks 返回用户任务状态列表（不分空投）
func (s *AirDropService) UserTasks(walletAddress string, page, size int) ([]TaskItemDTO, error) {
    addr := strings.ToLower(walletAddress)
    rows, err := ctx.Ctx.DB.Raw(`
        SELECT t.task_id, t.task_name, COALESCE(uts.user_status, 0) AS status
        FROM tasks t
        LEFT JOIN user_task_status uts ON uts.task_id = t.task_id AND uts.wallet_address = ?
        ORDER BY t.task_id ASC
        OFFSET ? LIMIT ?
    `, addr, (page-1)*size, size).Rows()
    if err != nil { return nil, err }
    defer rows.Close()
    var res []TaskItemDTO
    for rows.Next() {
        var it TaskItemDTO
        if err := rows.Scan(&it.TaskId, &it.TaskName, &it.Status); err == nil {
            res = append(res, it)
        }
    }
    return res, nil
}

// helpers
func nilToZero(v sql.NullString) string {
    if v.Valid { return v.String }
    return "0"
}

func nilToEmpty(v sql.NullString) string {
    if v.Valid { return v.String }
    return ""
}

func sprintf(format string, whereClause string, order string) string {
    return strings.Replace(strings.Replace(format, "%s", whereClause, 1), "%s", order, 1)
}
