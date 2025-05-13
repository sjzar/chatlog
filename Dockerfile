# 使用 Ubuntu 作为基础镜像（推荐最新 LTS 版本，如 22.04）
FROM ubuntu:22.04 as buildenv


# 设置环境变量（容器内生效）
ENV LANG=C.UTF-8 \
    GOPATH=/go \
    GO_VERSION=1.24.0 \
    WORKDIR=/app \ 
    CC_FOR_TARGET=x86_64-w64-mingw32-gcc \
    CGO_CFLAGS="-pthread"

# 安装基础工具和依赖
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    clang \
    llvm \
    libc6-dev \
    make \
    gcc \
    git \
    upx \
    mingw-w64 \
    gcc-aarch64-linux-gnu \
    wget \
    ca-certificates \
    tar \
    && rm -rf /var/lib/apt/lists/*

# 安装 Go 语言
RUN wget -O go.tar.gz "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" \
    && tar -C /usr/local -xzf go.tar.gz \
    && rm go.tar.gz \
    && ln -s /usr/local/go/bin/go /usr/bin/go

# 配置 Go 环境变量
ENV PATH="/usr/local/go/bin:${GOPATH}/bin:${PATH}" \
    GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct

# 安装 golangci-lint（代码检查工具）
RUN go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.6

# 设置工作目录（代码将挂载到此）
WORKDIR /app

# 启动时进入交互式 shell（方便执行 make 命令）
CMD ["bash"]
