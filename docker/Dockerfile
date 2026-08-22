FROM alpine:3.20

# WORKDIR 在容器内部设置工作目录 /app. 后面所有命令默认在这个文件夹执行
WORKDIR /app

RUN apk add --no-cache tzdata
ENV TZ=Asia/Shanghai
# `docker-compose.yml` 里构建上下文为 context: . 是项目根目录 所以此处不需要../
COPY config ./config
COPY static ./static

# EXPOSE 文档标记 告知他人该容器暴露 8080 端口 不会真正打开端口 真正的端口映射写在 compose
EXPOSE 8080

# 直接跑编译好的二进制文件
CMD ["./chatapp"]