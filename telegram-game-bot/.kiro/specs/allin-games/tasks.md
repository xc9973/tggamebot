# Implementation Plan: 梭哈游戏系统

## Overview

实现三种梭哈玩法：梭哈打劫、梭哈对决、梭哈骰子。

## Tasks

- [x] 1. 创建 AllIn 游戏核心模块
  - [x] 1.1 创建 `internal/game/allin/allin.go` 定义常量、结构体和接口
    - 定义 MinAllInBalance, AllInRobCooldown, AllInDiceCooldown, DuelTimeout 常量
    - 定义 AllInGame, DuelRequest, AllInResult, DuelResult, DiceResult 结构体
    - _Requirements: 1.1, 2.5, 3.1_

  - [x] 1.2 实现 AllInRob 梭哈打劫功能
    - 检查余额>=100
    - 检查冷却时间
    - 检查皇帝的新衣
    - 50%成功/失败逻辑
    - 转账金额=min(双方余额)
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

  - [x] 1.3 实现 AllInDice 梭哈骰子功能
    - 检查余额>=100
    - 检查冷却时间
    - 掷两个骰子(1-6)
    - >=7翻倍，<=6清零
    - 记录交易
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6_

  - [x] 1.4 实现对决系统 (CreateDuel, AcceptDuel, DeclineDuel)
    - CreateDuel: 创建待处理对决
    - AcceptDuel: 执行对决，50/50
    - DeclineDuel: 取消对决
    - 超时清理
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6_

- [x] 2. 创建 Handler 处理命令
  - [x] 2.1 创建 `internal/handler/allin.go` 处理命令
    - /shdj @target 命令
    - /duijue @target 命令
    - /shdice 命令
    - _Requirements: 4.1, 4.2, 4.5_

  - [x] 2.2 实现对决按钮回调处理
    - 接受按钮回调
    - 拒绝按钮回调
    - _Requirements: 4.3, 4.4_

- [-] 3. 注册命令和回调
  - [x] 3.1 在 bot.go 中注册新命令和回调
    - 注册 /shdj, /duijue, /shdice 命令
    - 注册对决按钮回调
    - _Requirements: 4.1-4.5_

- [x] 4. Checkpoint - 测试功能
  - 确保所有命令正常工作
  - 测试梭哈打劫、对决、骰子功能
