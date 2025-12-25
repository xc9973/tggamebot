# Implementation Plan: Go Telegram Game Bot

## Overview

使用 Go 语言完全重写 Telegram 游戏机器人，采用模块化架构，支持未来扩展新游戏。

## Tasks

- [x] 1. 项目初始化和基础架构
  - [x] 1.1 创建 Go 项目结构和 go.mod
    - 初始化 `telegram-game-bot` 目录
    - 添加依赖: telebot/v3, pgx/v5, viper, zerolog, rapid
    - _Requirements: 8.1, 10.1_
  - [x] 1.2 实现配置管理 (internal/config)
    - 使用 viper 加载 YAML 配置
    - 支持环境变量覆盖
    - _Requirements: 7.3, 8.3_
  - [x] 1.3 实现 PostgreSQL 连接池 (internal/pkg/db)
    - 使用 pgx/v5 连接池
    - 配置连接数、超时等参数
    - _Requirements: 8.3_

- [x] 2. 数据库迁移和模型
  - [x] 2.1 创建数据库迁移文件
    - 001_create_users.up.sql / down.sql
    - 002_create_transactions.up.sql / down.sql
    - 003_create_daily_stats_view.up.sql / down.sql
    - _Requirements: 8.1, 8.2, 8.4_
  - [x] 2.2 实现数据模型 (internal/model)
    - User, Transaction, DailyRank 结构体
    - _Requirements: 8.1, 8.2_

- [x] 3. Repository 层实现
  - [x] 3.1 实现 UserRepository
    - Create, GetByID, UpdateBalance, GetTopUsers
    - UpdateDailyClaim, CanClaimDaily
    - _Requirements: 1.1, 1.3, 1.5_
  - [x] 3.2 实现 TransactionRepository
    - Create, GetByUserID, GetDailyStats
    - _Requirements: 2.5, 11.2_
  - [x] 3.3 编写 Repository 单元测试
    - 使用 testcontainers 测试 PostgreSQL
    - _Requirements: 8.1, 8.2_

- [x] 4. Service 层实现
  - [x] 4.1 实现 AccountService
    - EnsureUser, GetBalance, UpdateBalance, ClaimDaily
    - _Requirements: 1.1, 1.2, 1.3, 1.4_
  - [x] 4.2 编写 Property Test: 每日签到资格
    - **Property 2: Daily Claim Eligibility**
    - **Validates: Requirements 1.3, 1.4**
  - [x] 4.3 实现 TransferService
    - Transfer 方法，包含验证逻辑
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_
  - [x] 4.4 编写 Property Test: 转账守恒和验证
    - **Property 4: Transfer Conservation**
    - **Property 5: Transfer Validation**
    - **Validates: Requirements 2.1, 2.2, 2.3, 2.4**
  - [x] 4.5 实现 RankingService
    - GetTopUsers, GetDailyWinners, GetDailyLosers
    - _Requirements: 1.5, 11.1, 11.2, 11.3_
  - [x] 4.6 编写 Property Test: 排行榜排序
    - **Property 3: Top Users Ordering**
    - **Property 14: Daily Ranking Calculation**
    - **Validates: Requirements 1.5, 11.2, 11.3, 11.5**

- [x] 5. Checkpoint - 基础服务完成
  - 确保所有测试通过，如有问题请询问用户

- [x] 6. 游戏框架实现
  - [x] 6.1 定义 Game 接口 (internal/game/interface.go)
    - Game, MultiPlayerGame 接口
    - _Requirements: 10.1, 10.3_
  - [x] 6.2 实现游戏注册表 (internal/game/registry.go)
    - Register, Get, List 方法
    - _Requirements: 10.2_
  - [x] 6.3 实现用户锁 (internal/pkg/lock)
    - 基于 sync.Map 的用户级锁
    - _Requirements: 9.1, 9.2_

- [x] 7. 骰子游戏实现
  - [x] 7.1 实现 DiceGame (internal/game/dice)
    - CalculatePayout 方法
    - 实现 Game 接口
    - _Requirements: 3.2, 3.3, 3.5_
  - [x] 7.2 编写 Property Test: 骰子赔付计算
    - **Property 6: Dice Payout Calculation**
    - **Validates: Requirements 3.2**

- [x] 8. 老虎机游戏实现
  - [x] 8.1 实现 SlotGame (internal/game/slot)
    - DecodeSlot, CalculatePayout 方法
    - 实现 Game 接口
    - _Requirements: 4.2, 4.4_
  - [x] 8.2 编写 Property Test: 老虎机计算
    - **Property 7: Slot Decode Correctness**
    - **Property 8: Slot Payout Calculation**
    - **Validates: Requirements 4.2, 4.4**

- [x] 9. 骰宝游戏实现
  - [x] 9.1 实现 SicBoCalculator (internal/game/sicbo/calculator.go)
    - IsTriple, CalculatePayout 方法
    - 单一数字和大小的赔率计算（固定100金币）
    - _Requirements: 5.3, 5.4, 5.5_
  - [x] 9.2 编写 Property Test: 骰宝赔付计算
    - **Property 9: SicBo Payout Calculation**
    - **Validates: Requirements 5.3, 5.4, 5.5**
  - [x] 9.3 实现 SicBoGame (internal/game/sicbo/sicbo.go)
    - 实现 MultiPlayerGame 接口
    - StartSession, PlaceBet, Settle 方法
    - 固定下注金额 100 金币
    - _Requirements: 5.1, 5.2, 5.7, 5.8_
  - [x] 9.4 编写 Property Test: 骰宝下注累加
    - **Property 10: SicBo Bet Accumulation**
    - **Validates: Requirements 5.8**
  - [x] 9.5 实现 SicBoKeyboard (internal/game/sicbo/keyboard.go)
    - 生成下注面板: [1][2][3][4][5][6] + [大][小]
    - 每个按钮点击下注 100 金币
    - _Requirements: 5.6_

- [x] 10. Checkpoint - 游戏逻辑完成
  - 确保所有测试通过，如有问题请询问用户

- [x] 11. Bot Handler 实现
  - [x] 11.1 实现 Bot 初始化 (internal/bot/bot.go)
    - 创建 telebot 实例
    - 注册所有 handler
    - _Requirements: 7.3_
  - [x] 11.2 实现中间件 (internal/bot/middleware.go)
    - 白名单检查中间件
    - 管理员权限中间件
    - _Requirements: 6.4, 7.1, 7.2_
  - [x] 11.3 编写 Property Test: 权限检查
    - **Property 11: Admin Permission Check**
    - **Property 12: Whitelist Enforcement**
    - **Validates: Requirements 6.4, 7.1**
  - [x] 11.4 实现账户 Handler (internal/handler/account.go)
    - /start, /balance, /my, /daily, /top 命令
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_
  - [x] 11.5 实现转账 Handler (internal/handler/transfer.go)
    - /pay 命令
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_
  - [x] 11.6 实现管理员 Handler (internal/handler/admin.go)
    - /admin_add, /admin_sub, /admin_set 命令
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_
  - [x] 11.7 实现排行榜 Handler (internal/handler/ranking.go)
    - /daily_top 命令
    - _Requirements: 11.1, 11.3_
  - [x] 11.8 实现游戏 Handler (internal/handler/game.go)
    - /dice, /slot 命令
    - 骰宝相关命令和回调
    - _Requirements: 3.1, 4.1, 5.1, 5.5_

- [x] 12. 并发控制集成
  - [x] 12.1 在 Handler 中集成用户锁
    - 余额操作前获取锁
    - 操作完成后释放锁
    - _Requirements: 9.1, 9.2_
  - [x] 12.2 编写 Property Test: 并发安全
    - **Property 13: Concurrent Balance Safety**
    - **Validates: Requirements 9.1, 9.3**

- [x] 13. 入口和部署
  - [x] 13.1 实现 main.go (cmd/bot/main.go)
    - 加载配置
    - 初始化数据库
    - 运行迁移
    - 启动 Bot
    - _Requirements: 8.4_
  - [x] 13.2 创建 Dockerfile
    - 多阶段构建
    - 最小化镜像
  - [x] 13.3 创建 docker-compose.yml
    - Bot 服务
    - PostgreSQL 服务
    - _Requirements: 8.1_

- [x] 14. Final Checkpoint - 完整测试
  - 确保所有测试通过
  - 手动测试所有命令
  - 如有问题请询问用户

## Notes

- Tasks marked with `*` are optional property-based tests
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Game interface design allows easy addition of new games in the future
