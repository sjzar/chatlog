# MCP 集成指南

## 目录
- [MCP 集成指南](#mcp-集成指南)
  - [目录](#目录)
  - [前期准备](#前期准备)
    - [mcp-proxy](#mcp-proxy)
  - [ChatWise](#chatwise)
  - [Cherry Studio](#cherry-studio)
  - [Claude Desktop](#claude-desktop)
  - [Monica Code](#monica-code)


## 前期准备

运行 `chatlog`，完成数据解密并开启 HTTP 服务

### mcp-proxy
如果遇到不支持 `SSE` 的客户端，可以尝试使用 `mcp-proxy` 将 `stdio` 的请求转换为 `SSE`。  

项目地址：https://github.com/sparfenyuk/mcp-proxy

安装方式：
```shell
# 使用 uv 工具安装，也可参考项目文档的其他安装方式
uv tool install mcp-proxy

# 查询 mcp-proxy 的路径，后续可直接使用该路径
which mcp-proxy
/Users/sarv/.local/bin/mcp-proxy
```

## ChatWise

- 官网：https://chatwise.app/
- 使用方式：MCP SSE
- 注意事项：使用 ChatWise 的 MCP 功能需要 Pro 权限

1. 在 `设置 - 工具` 下新建 `SSE 请求` 工具

![chatwise-1](https://github.com/user-attachments/assets/87e40f39-9fbc-4ff1-954a-d95548cde4c2)

1. 在 URL 中填写 `http://127.0.0.1:5030/sse`，并勾选 `自动执行工具`，点击 `查看工具` 即可检查连接 `chatlog` 是否正常

![chatwise-2](https://github.com/user-attachments/assets/8f98ef18-8e6c-40e6-ae78-8cd13e411c36)

3. 返回主页，选择支持 MCP 调用的模型，打开 `chatlog` 工具选项

![chatwise-3](https://github.com/user-attachments/assets/ea2aa178-5439-492b-a92f-4f4fc08828e7)

4. 测试功能是否正常

![chatwise-4](https://github.com/user-attachments/assets/8f82cb53-8372-40ee-a299-c02d3399403a)

## Cherry Studio

- 官网：https://cherry-ai.com/
- 使用方式：MCP SSE

1. 在 `设置 - MCP 服务器` 下点击 `添加服务器`，输入名称为 `chatlog`，选择类型为 `服务器发送事件(sse)`，填写 URL 为 `http://127.0.0.1:5030/sse`，点击 `保存`。（注意：点击保存前不要先点击左侧的开启按钮）

![cherry-1](https://github.com/user-attachments/assets/93fc8b0a-9d95-499e-ab6c-e22b0c96fd6a)

2. 选择支持 MCP 调用的模型，打开 `chatlog` 工具选项

![cherry-2](https://github.com/user-attachments/assets/4e5bf752-2eab-4e7c-b73b-1b759d4a5f29)

3. 测试功能是否正常

![cherry-3](https://github.com/user-attachments/assets/c58a019f-fd5f-4fa3-830a-e81a60f2aa6f)

## Claude Desktop

- 官网：https://claude.ai/download
- 使用方式：mcp-proxy
- 参考资料：https://modelcontextprotocol.io/quickstart/user#2-add-the-filesystem-mcp-server

1. 请先参考 [mcp-proxy](#mcp-proxy) 安装 `mcp-proxy`

2. 进入 Claude Desktop `Settings - Developer`，点击 `Edit Config` 按钮，这样会创建一个 `claude_desktop_config.json` 配置文件，并引导你编辑该文件

3. 编辑 `claude_desktop_config.json` 文件，配置名称为 `chatlog`，command 为 `mcp-proxy` 的路径，args 为 `http://127.0.0.1:5030/sse`，如下所示：

```json
{
  "mcpServers": {
    "chatlog": {
      "command": "/Users/sarv/.local/bin/mcp-proxy",
      "args": [
        "http://localhost:5030/sse"
      ]
    }
  },
  "globalShortcut": ""
}
```

4. 保存 `claude_desktop_config.json` 文件，重启 Claude Desktop，可以看到 `chatlog` 已经添加成功

![claude-1](https://github.com/user-attachments/assets/f4e872cc-e6c1-4e24-97da-266466949cdf)

5. 测试功能是否正常

![claude-2](https://github.com/user-attachments/assets/832bb4d2-3639-4cbc-8b17-f4b812ea3637)


## Monica Code

- 官网：https://monica.im/en/code
- 使用方式：mcp-proxy
- 参考资料：https://github.com/Monica-IM/Monica-Code/blob/main/Reference/config.md#modelcontextprotocolserver

1. 请先参考 [mcp-proxy](#mcp-proxy) 安装 `mcp-proxy`

2. 在 vscode 插件文件夹（`~/.vscode/extensions`）下找到 Monica Code 的目录，编辑 `config_schema.json` 文件。将 `experimental - modelContextProtocolServer` 中 `transport` 设置为如下内容：

```json
{
  "experimental": {
    "type": "object",
    "title": "Experimental",
    "description": "Experimental properties are subject to change.",
    "properties": {
      "modelContextProtocolServer": {
        "type": "object",
        "properties": {
          "transport": {
            "type": "stdio",
            "command": "/Users/sarv/.local/bin/mcp-proxy",
            "args": [
              "http://localhost:5030/sse"
            ]
          }
        },
        "required": [
          "transport"
        ]
      }
    }
  }
}
```

3. 重启 vscode，可以看到 `chatlog` 已经添加成功

![monica-1](https://github.com/user-attachments/assets/8d0a96f2-ed05-48aa-a99a-06648ae1c500)

4. 测试功能是否正常

![monica-2](https://github.com/user-attachments/assets/054e0a30-428a-48a6-9f31-d2596fb8f743)

