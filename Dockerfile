# 使用官方的Alpine Linux镜像作为基础镜像
FROM alpine:latest

# 设置工作目录
WORKDIR /app

# 将编译好的可执行文件复制到容器中
COPY go_ddns /app
# 复制配置文件
COPY config.toml /app

# 暴露容器的8080端口
EXPOSE 8080

# 启动程序
CMD ["./go_ddns"]