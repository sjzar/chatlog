## (一) 构建镜像并挂载代码

### 1. 构建镜像
```bash
# 假设代码在当前目录，镜像命名为 chatlog-builder
docker build -t chatlog-builder .
```

### 2. 运行容器并挂载代码
```bash
# -v 将本地代码目录挂载到容器内的 /app（与 Dockerfile 中 WORKDIR 一致）
# -it 启用交互式终端
docker run -it \
  -v $(pwd):/app \  # 挂载当前目录（替换为你的代码路径）
  --rm \  # 容器退出后自动删除
  chatlog-builder
```

此时进入容器内的 `/app` 目录，即为你的项目代码（可直接修改本地文件，容器内实时同步）。

## (二) 在容器内执行编译命令
#### 1. 基础操作（与本地一致）
```bash
# 初始化依赖
make tidy

# 本地编译（Linux amd64，容器默认架构）
make build  # 生成 bin/chatlog

# 完整流程（清理、检查、测试、编译）
make all

# 多平台交叉编译（生成 Windows/macOS 二进制）
make crossbuild  # 结果在 bin/ 目录
```

#### 2. 启用 UPX 压缩（可选）
```bash
# 在容器内执行时设置环境变量
ENABLE_UPX=1 make crossbuild
```


### 关键设计说明
#### 1. **依赖安装策略**
- **系统依赖**（如 `mingw-w64`、`upx`）通过 `apt` 安装，确保交叉编译环境完整。
- **Go 工具链**直接从官网下载二进制包，避免版本冲突（通过 `GO_VERSION` 可灵活调整）。
- **golangci-lint**通过 `go install` 安装，保证与项目兼容的最新版本。

#### 2. **代码挂载与实时修改**
- 使用 `-v $(pwd):/app` 将本地代码目录挂载到容器，修改本地文件后无需重启容器，直接执行 `make` 即可生效。
- 容器内工作目录固定为 `/app`，与 Makefile 预期的项目根目录一致。

#### 3. **多平台编译支持**
- `mingw-w64` 提供 Windows 交叉编译所需的工具链（如 `x86_64-w64-mingw32-gcc`），Makefile 中 `GOOS=windows` 时自动调用。
- Darwin（macOS）平台无需额外工具，Go 原生支持跨平台编译（`GOOS=darwin`）。

#### 4. **镜像轻量化**
- 使用 `--no-install-recommends` 减少不必要的包，`rm -rf /var/lib/apt/lists/*` 清理缓存。
- Go 安装包直接解压到 `/usr/local/go`，避免通过源码编译，提升构建速度。


### 五、进阶优化（可选）
#### 1. **非 root 用户运行（安全增强）**
```dockerfile
# 在 Dockerfile 中添加以下内容（建议生产环境使用）
RUN useradd -m -s /bin/bash developer \
    && chown -R developer:developer /app \
    && chown -R developer:developer /go
USER developer
```

#### 2. **多阶段构建（仅导出编译结果）**
若只需生成二进制文件（不用于开发），可通过多阶段构建减小最终镜像体积：
```dockerfile
# 第一阶段：构建环境（同上文）
FROM ubuntu:22.04 as buildenv
...（安装依赖、Go、golangci-lint）...

# 第二阶段：导出编译结果（可选，用于生成最终制品）
FROM scratch
COPY --from=buildenv /app/bin/ /bin/
ENTRYPOINT ["/bin/chatlog"]
```

#### 3. **指定镜像架构（支持多平台镜像）**
若需构建跨架构镜像（如 arm64 环境运行 amd64 镜像），使用 Docker Buildx：
```bash
# 启用 Buildx
docker buildx create --use
# 构建多平台镜像
docker buildx build --platform linux/amd64,linux/arm64 -t chatlog-builder:multi .
```


### 六、总结
通过以上 Docker 镜像，可在任意支持 Docker 的环境（如 Linux、Windows Subsystem for Linux、macOS）中：
1. 实时修改本地代码并在容器内生效。
2. 执行 `make` 命令完成代码检查、测试和多平台编译。
3. 避免本地环境配置繁琐，确保依赖一致性。

**最终镜像使用流程**：  
```bash
# 构建镜像
docker build -t chatlog-builder .
# 运行容器（挂载代码）
docker run -it -v /path/to/chatlog:/app chatlog-builder
# 容器内执行编译
make crossbuild
```

编译生成的多平台二进制文件（如 `bin/chatlog_darwin_amd64`、`bin/chatlog_windows_amd64.exe`）会直接保存在本地代码目录的 `bin/` 中，无需从容器中拷贝。
