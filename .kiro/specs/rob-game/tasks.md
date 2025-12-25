# Implementation Plan: Rob Game (打劫游戏)

## Overview

为 Go Telegram 游戏机器人添加打劫游戏功能，允许玩家通过 `/dajie` 命令打劫其他用户的金币。

## Tasks

- [x] 1. 打劫游戏核心逻辑
  - [x] 1.1 创建 RobGame 结构 (internal/game/rob/rob.go)
    - 定义 RobGame, ProtectionState, RobResult 结构体
    - 实现内存存储: protection map, cooldowns map
    - _Requirements: 3.1, 4.1_
  - [x] 1.2 实现金额生成
    - generateAmount() 生成 10-1000 随机金额
    - _Requirements: 2.1_
  - [x] 1.3 实现打劫验证 CanRob()
    - 检查自己打劫自己
    - 检查目标用户是否注册
    - 检查冷却时间 (21秒)
    - 检查保护期
    - _Requirements: 1.3, 1.4, 3.3, 4.1_
  - [x] 1.4 实现打劫执行 Rob()
    - 获取用户锁
    - 计算实际打劫金额 (min(生成金额, 目标余额))
    - 转移金币
    - 记录交易
    - 更新保护期状态
    - _Requirements: 2.2, 2.3, 3.1, 3.2, 5.1, 5.2_

- [x] 2. 保护期机制
  - [x] 2.1 实现 IsProtected() 检查保护期
    - 返回是否在保护期及剩余时间
    - _Requirements: 3.3_
  - [x] 2.2 实现保护期激活逻辑
    - 连续被打劫 3 次后激活 30 分钟保护期
    - 保护期结束后重置计数
    - _Requirements: 3.1, 3.2, 3.5_

- [x] 3. 冷却机制
  - [x] 3.1 实现 GetCooldown() 获取冷却剩余时间
    - _Requirements: 4.2_
  - [x] 3.2 实现冷却设置和检查
    - 21 秒冷却时间
    - _Requirements: 4.1, 4.3_

- [x] 4. Property Tests
  - [x] 4.1 编写 Property Test: 金额范围
    - **Property 1: Robbery Amount Range**
    - 验证生成金额在 [10, 1000] 范围内
    - **Validates: Requirements 2.1**
  - [x] 4.2 编写 Property Test: 打劫验证
    - **Property 1: Robbery Validation**
    - 验证自己不能打劫自己
    - **Validates: Requirements 1.3**
  - [x] 4.3 编写 Property Test: 保护期机制
    - **Property 3: Protection Mechanism**
    - 验证连续 3 次被打劫后激活保护期
    - **Validates: Requirements 3.1, 3.2**
  - [x] 4.4 编写 Property Test: 冷却时间
    - **Property 4: Cooldown Enforcement**
    - 验证 21 秒冷却时间
    - **Validates: Requirements 4.1**

- [x] 5. Handler 集成
  - [x] 5.1 添加 /dajie Handler (internal/handler/game.go)
    - 解析回复消息或 @username
    - 调用 RobGame.Rob()
    - 发送结果消息
    - _Requirements: 1.1, 1.2, 2.4_
  - [x] 5.2 更新 Bot 注册
    - 在 bot.go 中注册 /dajie 命令
    - _Requirements: 1.1_

- [x] 6. 交易类型更新
  - [x] 6.1 添加交易类型常量 (internal/model/transaction.go)
    - TxTypeRob = "rob"
    - TxTypeRobbed = "robbed"
    - _Requirements: 5.1, 5.2_
  - [x] 6.2 更新排行榜查询
    - 确保 rob/robbed 交易计入每日排行
    - _Requirements: 5.3_

- [x] 7. Final Checkpoint
  - 运行所有测试
  - 手动测试 /dajie 命令
  - 验证保护期和冷却机制

## Notes

- 保护期和冷却状态存储在内存中，重启后重置
- 打劫金额上限为目标用户余额
- 冷却时间只对打劫者生效，不影响被打劫者
