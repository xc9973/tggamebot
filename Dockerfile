FROM python:3.11-slim

WORKDIR /app

# 安装依赖
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# 复制源代码
COPY src/ ./src/

# 创建数据目录
RUN mkdir -p /app/data /app/config

# 设置环境变量
ENV BOT_CONFIG_PATH=/app/config/config.json

# 运行机器人
CMD ["python", "-m", "src.main"]
