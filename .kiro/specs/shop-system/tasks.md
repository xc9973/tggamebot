# Implementation Plan: Shop System (Updated)

## Overview

更新商店系统，所有道具改为次数限制型，新增4个道具（钝刀、大宝剑、紫金袈裟、皇帝的新衣），实现每日购买限制和防御优先级。

## Tasks

- [x] 1. 创建道具模型和配置 (已完成)

- [x] 2. 创建数据库迁移和 Repository (已完成)

- [x] 3. 创建商店服务 (已完成)

- [x] 4. 创建商店 Handler 和键盘 (已完成)

- [x] 5. 集成打劫游戏 (已完成)

- [x] 6. 注册 Handler 和依赖注入 (已完成)

- [x] 7. Checkpoint - 测试商店功能 (已完成)

---

## 新增任务：商品属性修改和新商品

- [x] 8. 更新道具配置（次数限制型）
  - [x] 8.1 修改 `internal/shop/items.go` 更新现有道具
    - 移除 Duration 字段，改用 UseCount 字段
    - 手铐：UseCount=1，DailyLimit=5
    - 保护罩：UseCount=10，DailyLimit=2
    - 荆棘刺甲：UseCount=5
    - 饮血剑：UseCount=10
    - _Requirements: 2.1-2.3, 3.1-3.3, 4.1-4.2, 5.1-5.2_
  - [x] 8.2 添加新道具类型常量和配置
    - 钝刀 (blunt_knife): 1000金币，UseCount=10，BypassDefense=true
    - 大宝剑 (great_sword): 10000金币，UseCount=3，DailyLimit=1，BypassDefense=true
    - 紫金袈裟 (golden_cassock): 10000金币，UseCount=3
    - 皇帝的新衣 (emperor_clothes): 5000金币，UseCount=3，ImmuneBypass=true
    - _Requirements: 6.1-6.2, 7.1-7.3, 8.1-8.2, 9.1-9.2_
  - [x] 8.3 更新 GetAllItems() 返回8个商品
    - _Requirements: 1.1, 1.2_

- [x] 9. 更新数据库和 Repository
  - [x] 9.1 修改 user_items 表结构
    - 将 quantity 改为 use_count
    - 移除 user_effects 表（不再需要时间限制）
    - _Requirements: 3.7, 4.5, 5.5, 6.6, 7.7, 8.5, 9.6_
  - [x] 9.2 添加 daily_purchases 表
    - _Requirements: 12.1, 12.2_
  - [x] 9.3 更新 `internal/repository/inventory.go`
    - AddItem 改为增加 use_count
    - 添加 GetUseCount 方法
    - 添加 DecrementUseCount 方法
    - 添加 GetDailyPurchaseCount 方法
    - 添加 IncrementDailyPurchase 方法
    - _Requirements: 3.6, 12.1, 12.3_
  - [x] 9.4 编写 Property Test: Use Count Decrement
    - **Property 3: Use Count Decrement**
    - **Validates: Requirements 3.6, 3.7, 4.4, 4.5, 5.4, 5.5, 6.5, 6.6, 7.6, 7.7, 8.4, 8.5, 9.5, 9.6**

- [x] 10. 更新商店服务
  - [x] 10.1 更新 `internal/service/shop.go`
    - PurchaseItem 添加每日限制检查
    - 添加 GetEffectUseCount 方法
    - 添加 DecrementUseCount 方法
    - 添加 HasEmperorClothes 方法
    - 添加 HasBluntKnife 方法
    - 添加 HasGreatSword 方法
    - 添加 HasGoldenCassock 方法
    - 添加 RemoveDefensiveItems 方法
    - _Requirements: 6.3, 7.4, 8.3, 9.3, 12.3, 12.4_
  - [x] 10.2 编写 Property Test: Daily Purchase Limit
    - **Property 2: Daily Purchase Limit Enforcement**
    - **Validates: Requirements 2.3, 2.9, 3.3, 3.8, 7.3, 7.8, 12.1, 12.3, 12.4**

- [x] 11. 实现皇帝的新衣效果（最高优先级防御）
  - [x] 11.1 更新 `internal/game/rob/rob.go`
    - 在防御检查中首先检查皇帝的新衣
    - 皇帝的新衣免疫所有攻击（包括钝刀、大宝剑）
    - _Requirements: 9.4, 9.5_
  - [x] 11.2 编写 Property Test: Emperor Clothes Immunity
    - **Property 4: Emperor Clothes Immunity**
    - **Validates: Requirements 9.4, 9.5**

- [x] 12. 实现钝刀效果
  - [x] 12.1 更新 `internal/game/rob/rob.go`
    - 检查钝刀效果，无视保护罩和荆棘刺甲（但不能无视皇帝的新衣）
    - 钝刀生效时打劫金额限制为1-100随机
    - _Requirements: 6.4, 6.5_
  - [x] 12.2 编写 Property Test: Blunt Knife Amount Limit
    - **Property 6: Blunt Knife Amount Limit**
    - **Validates: Requirements 6.5**

- [x] 13. 实现大宝剑效果
  - [x] 13.1 更新 `internal/game/rob/rob.go`
    - 检查大宝剑效果，无视保护罩和荆棘刺甲（但不能无视皇帝的新衣）
    - 大宝剑生效时 0.01% 概率打劫90%金币
    - _Requirements: 7.5, 7.6_
  - [x] 13.2 编写 Property Test: Great Sword Critical Hit
    - **Property 7: Great Sword Critical Hit**
    - **Validates: Requirements 7.6**

- [x] 14. 实现紫金袈裟效果
  - [x] 14.1 更新 `internal/game/rob/rob.go`
    - 检查目标是否有紫金袈裟
    - 触发时移除攻击者的保护罩和荆棘刺甲
    - _Requirements: 8.4_
  - [x] 14.2 编写 Property Test: Golden Cassock Defense Removal
    - **Property 8: Golden Cassock Defense Removal**
    - **Validates: Requirements 8.4**

- [x] 15. 更新商店界面
  - [x] 15.1 更新 `internal/shop/keyboard.go`
    - BuildShopPanel 显示8个商品按钮
    - 显示使用次数和每日限购信息
    - _Requirements: 1.1, 1.2_
  - [x] 15.2 更新 `internal/handler/shop.go`
    - 购买时检查每日限制
    - 显示每日限购错误信息
    - _Requirements: 2.9, 3.8, 7.8_

- [x] 16. 更新背包显示
  - [x] 16.1 更新 `internal/handler/shop.go` HandleBag
    - 显示道具剩余使用次数
    - _Requirements: 11.2_

- [x] 17. Checkpoint - 测试新功能
  - 确保所有测试通过
  - 测试每日购买限制
  - 测试道具使用次数递减
  - 测试皇帝的新衣免疫所有攻击
  - 测试钝刀无视防御效果
  - 测试大宝剑无视防御和暴击效果
  - 测试紫金袈裟移除攻击者防御效果
  - 测试防御优先级顺序

## Notes

- Tasks marked with `*` are optional property-based tests
- 共8个商品：手铐、保护罩、荆棘刺甲、饮血剑、钝刀、大宝剑、紫金袈裟、皇帝的新衣
- 所有道具都是次数限制型，用完即失效
- 防御优先级：皇帝的新衣 > 保护罩 > 荆棘刺甲
- 皇帝的新衣是唯一能免疫钝刀和大宝剑的道具
- 价格配置集中在 items.go，方便后期调整
