# Telegram 游戏机器人

一个功能丰富的 Telegram 群组游戏机器人，支持虚拟积分系统、多种概率游戏和用户互动功能。

## 功能特性

- 🎮 **虚拟积分系统** - 每位用户拥有独立账户，初始 1000 金币
- 📅 **每日签到** - 每 24 小时可领取 500 金币
- 💸 **用户转账** - 支持用户间转账（5% 手续费）
- 🏆 **财富排行榜** - 查看群内最富有的用户
- 🎲 **骰子游戏** - 简单的概率游戏
- 🎰 **老虎机游戏** - 高赔率概率游戏
- 🃏 **21点游戏** - 策略性纸牌游戏
- 🎲 **骰宝游戏** - 多人参与的骰子游戏
- 👮 **管理员功能** - 金币管理和账户重置

## 安装

### 环境要求

- Python 3.10+
- pip

### 安装步骤

1. 克隆项目
```bash
git clone <repository-url>
cd telegram-game-bot
```

2. 安装依赖
```bash
pip install -r requirements.txt
```

3. 配置机器人
```bash
cp config/config.example.json config/config.json
```

4. 编辑 `config/config.json`，填写以下信息：
   - `bot_token`: 从 [@BotFather](https://t.me/BotFather) 获取
   - `admin_ids`: 管理员的 Telegram ID（可通过 [@userinfobot](https://t.me/userinfobot) 获取）
   - `database_path`: 数据库文件路径（默认 `data/bot.db`）

5. 启动机器人
```bash
python -m src.main
```

## Docker 部署

### 构建镜像
```bash
docker build -t telegram-game-bot .
```

### 运行容器
```bash
docker run -d \
  --name game-bot \
  -v $(pwd)/config:/app/config \
  -v $(pwd)/data:/app/data \
  telegram-game-bot
```

## 命令列表

### 基础命令

| 命令 | 说明 |
|------|------|
| `/start` | 初始化账户，显示帮助信息 |
| `/balance` 或 `/my` | 查询当前余额 |
| `/daily` | 每日签到领取 500 金币 |
| `/top` | 查看财富排行榜 TOP 10 |

### 转账命令

| 命令 | 说明 |
|------|------|
| `/pay @用户名 金额` | 向指定用户转账（5% 手续费） |

### 游戏命令

| 命令 | 说明 |
|------|------|
| `/dice 金额` | 骰子游戏 |
| `/slot 金额` | 老虎机游戏 |
| `/bj 金额` | 21点游戏 |
| `/sicbo` | 开始骰宝游戏 |
| `/bet single <数字> <金额>` | 骰宝：押单一数字 (1-6) |
| `/bet pair <数字1> <数字2> <金额>` | 骰宝：押两数组合 |
| `/bet sum <总和> <金额>` | 骰宝：押总和 (4-17) |
| `/bet big <金额>` | 骰宝：押大 (11-17) |
| `/bet small <金额>` | 骰宝：押小 (4-10) |
| `/roll` | 骰宝：开骰子 |
| `/sicbo_status` | 查看骰宝游戏状态 |
| `/mybets` | 查看我的骰宝押注 |

### 管理员命令

| 命令 | 说明 |
|------|------|
| `/admin_add @用户名 金额` | 向用户添加金币 |
| `/admin_remove @用户名 金额` | 从用户扣除金币 |
| `/admin_reset @用户名` | 重置用户账户 |

## 游戏规则

### 🎲 骰子游戏
- 点数 1-3：输掉本金
- 点数 4-5：赢得 1 倍本金
- 点数 6：赢得 2 倍本金

### 🎰 老虎机游戏
- 三个图案一致：赢得 10 倍本金
- 两个图案一致：赢得 2 倍本金
- 三个图案不一致：输掉本金

### 🃏 21点游戏
- 目标：手牌点数尽量接近 21 点但不超过
- A 可算 1 点或 11 点
- J、Q、K 算 10 点
- Blackjack（首两张 21 点）：赢得 1.5 倍本金
- 可选操作：要牌、停牌、加倍

### 🎲 骰宝游戏

骰宝是一个多人参与的骰子游戏，使用三个骰子。玩家可以在多种下注区域进行押注。

**游戏流程：**
1. 使用 `/sicbo` 开始游戏
2. 60 秒内玩家可以下注
3. 使用 `/roll` 或等待超时自动开骰子
4. 系统自动结算所有押注

**下注类型和赔率：**

| 下注类型 | 命令示例 | 赔率 |
|----------|----------|------|
| 单一数字 | `/bet single 3 100` | 1个匹配 1:1，2个匹配 2:1，3个匹配 3:1 |
| 两数组合 | `/bet pair 3 5 100` | 5:1 |
| 总和 4/17 | `/bet sum 4 100` | 60:1 |
| 总和 5/16 | `/bet sum 5 100` | 30:1 |
| 总和 6/15 | `/bet sum 6 100` | 17:1 |
| 总和 7/14 | `/bet sum 7 100` | 12:1 |
| 总和 8/13 | `/bet sum 8 100` | 8:1 |
| 总和 9/10/11/12 | `/bet sum 10 100` | 6:1 |
| 大 (11-17) | `/bet big 100` | 1:1 |
| 小 (4-10) | `/bet small 100` | 1:1 |

**特殊规则：**
- ⚠️ **围骰（三个相同）**：总和押注和大小押注庄家通吃
- 同一玩家可以同时下多个注
- 下注金额立即从余额扣除

## 配置说明

```json
{
  "bot_token": "YOUR_BOT_TOKEN_HERE",
  "database_path": "data/bot.db",
  "admin_ids": [123456789]
}
```

| 配置项 | 说明 |
|--------|------|
| `bot_token` | Telegram Bot Token |
| `database_path` | SQLite 数据库文件路径 |
| `admin_ids` | 管理员 Telegram ID 列表 |

## 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `BOT_CONFIG_PATH` | 配置文件路径 | `config/config.json` |

## 开发

### 运行测试
```bash
pytest
```

### 运行属性测试
```bash
pytest tests/ -v
```

## 许可证

MIT License
