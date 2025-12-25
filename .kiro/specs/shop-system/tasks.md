# Implementation Plan: Shop System

## Overview

实现商店系统，包括道具定义、数据库存储、购买流程、手铐使用、以及与打劫游戏的集成。

## Tasks

- [x] 1. 创建道具模型和配置
  - [x] 1.1 创建 `internal/shop/items.go` 定义道具类型和配置
    - 定义 ItemType 常量
    - 定义 ItemConfig 结构体
    - 定义 ShopItems 配置 map
    - _Requirements: 2.1, 3.1, 3.2, 4.1, 4.2, 5.1, 5.2_

- [x] 2. 创建数据库迁移和 Repository
  - [x] 2.1 在 `cmd/bot/main.go` 添加数据库迁移
    - 创建 user_items 表
    - 创建 user_effects 表
    - 创建 handcuff_locks 表
    - _Requirements: 6.1, 6.2_
  - [x] 2.2 创建 `internal/repository/inventory.go`
    - AddItem / GetItemCount / DecrementItem
    - AddEffect / GetActiveEffects / HasActiveEffect
    - AddHandcuffLock / IsHandcuffed / CleanExpiredLocks
    - _Requirements: 2.3, 3.3, 4.3, 5.3_

- [x] 3. 创建商店服务
  - [x] 3.1 创建 `internal/service/shop.go`
    - PurchaseItem 方法（扣款+添加道具）
    - UseHandcuff 方法（消耗手铐+锁定目标）
    - GetUserInventory 方法
    - HasShield / HasThornArmor / HasBloodthirstSword 方法
    - IsHandcuffed 方法
    - _Requirements: 1.4, 1.6, 2.2, 2.3, 2.7_
  - [ ]* 3.2 编写 Property Test: Purchase Transaction Integrity
    - **Property 1: Purchase Transaction Integrity**
    - **Validates: Requirements 1.4, 1.5, 1.6**

- [x] 4. 创建商店 Handler 和键盘
  - [x] 4.1 创建 `internal/shop/keyboard.go`
    - BuildShopPanel 构建商店主面板
    - BuildConfirmPanel 构建确认面板
    - _Requirements: 1.1, 1.2, 1.3_
  - [x] 4.2 创建 `internal/handler/shop.go`
    - HandleShopStart 处理私聊 /start
    - HandleShopCallback 处理按钮回调
    - HandleBag 处理 /bag 命令
    - _Requirements: 1.1, 1.3, 1.4, 1.5, 7.1, 7.2, 7.3_
  - [x] 4.3 创建 `internal/handler/handcuff.go`
    - HandleHandcuff 处理 /handcuff 命令
    - _Requirements: 2.3, 2.6_

- [-] 5. 集成打劫游戏
  - [x] 5.1 修改 `internal/game/rob/rob.go`
    - 添加 ShopService 依赖
    - CanRob 检查攻击方是否被手铐锁定
    - CanRob 检查目标是否有保护罩
    - DetermineOutcome 检查饮血剑效果（80%成功率）
    - Rob 成功后检查荆棘刺甲效果（扣双倍）
    - _Requirements: 2.4, 3.4, 4.4, 5.4_
  - [ ]* 5.2 编写 Property Test: Shield Protection Effect
    - **Property 4: Shield Protection Effect**
    - **Validates: Requirements 3.4**
  - [ ]* 5.3 编写 Property Test: Handcuff Lock Effect
    - **Property 7: Handcuff Lock Effect**
    - **Validates: Requirements 2.4**

- [x] 6. 注册 Handler 和依赖注入
  - [x] 6.1 修改 `internal/bot/bot.go`
    - 添加 ShopService 依赖
    - 添加 ShopHandler 依赖
    - 注册 /start (私聊), /handcuff, /bag 命令
    - 注册商店回调处理
    - _Requirements: 1.1, 2.3, 7.1_
  - [x] 6.2 修改 `cmd/bot/main.go`
    - 初始化 InventoryRepository
    - 初始化 ShopService
    - 传递依赖到 Bot
    - _Requirements: All_

- [ ] 7. Checkpoint - 测试商店功能
  - 确保所有测试通过
  - 测试私聊 /start 显示商店
  - 测试购买流程
  - 测试 /bag 显示背包
  - 测试 /handcuff 使用手铐
  - 测试打劫游戏集成

## Notes

- Tasks marked with `*` are optional property-based tests
- 商店只在私聊中显示，群聊 /start 保持原有行为
- 手铐是唯一需要主动使用的道具，其他道具购买后自动生效
- 过期效果通过查询时检查 expires_at 来处理，不需要后台清理任务
