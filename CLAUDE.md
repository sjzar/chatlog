Always respond in 中文

# 代码设计上的要求
- 各部分功能尽量低耦合，高内聚。
- 各个函数代码做好注释。
- 有完备的日志系统，方便后期调试确认问题。
- 避免写冗余代码，提高项目的可复用性。
- 尽力遵循工程上的最佳实践。适当使用文件夹来增加整个项目的可读性，但不要添加过多无关文件。
- 及时完善 .gitignore 文件。
- 涉及多平台运行时，请考虑文件编码问题。尤其是 requirements.txt 文件。
- 如果是 python 项目，记得配置虚拟环境并在虚拟环境中运行代码。
- 尽量保证编程语言的唯一性，避免多种编程语言混杂,比如 C++ 和 python 混杂。
- 设计项目时需要考虑全平台的兼容性，注意文件路径和文本编码。

# 工作流程  
- 完成一系列代码修改后，务必进行类型检查  
- 出于性能考虑，优先运行单个测试，而非整个测试套件
- 所有 test 项目请置于 tests 文件夹里面运行,适当创建子文件夹以提升项目可读性。禁止在其他文件夹里放 test 脚本。


# 项目文档编写指南 (最佳实践)

本文档旨在为本项目的文档编写提供一套标准和指南，以确保文档的清晰性、一致性，并使其易于长期维护。我们遵循 **“代码即文档，文档即网站”** 的核心理念。

---

## 1. 核心理念

我们的文档系统分为两大支柱：

1.  **代码内文档 (Docstrings)**: 直接写在 Python 代码中的文档，作为 API 功能的 **唯一真实来源 (Single Source of Truth)**。
2.  **项目级文档 (`/docs` 文件夹)**: 用于存放高级、概览性的说明文档，如教程、架构设计等。

这两大支柱将通过自动化工具 **Sphinx** 合并，最终生成一个统一的、可在线浏览的文档网站。

---

## 2. 项目结构

所有文档相关的内容都遵循以下目录结构：
```

your_project/

├── docs/                      # 存放项目级文档的目录

│   ├── index.md             # 文档网站首页 (必须)

│   ├── installation.md      # 安装与环境配置指南

│   ├── quickstart.md        # 快速上手教程

│   ├── architecture.md      # 核心架构与设计理念

│   └── ...                  # 其他高级文档

│

├── src/

│   └── your_package/

│       ├── init.py

│       ├── module_a.py        # 每个函数/类/方法都必须有 Docstring

│       └── component_b/

│           ├── init.py

│           └── feature.py     # ...同样需要有 Docstring

│

├── pyproject.toml             # 项目配置文件

└── README.md                  # 简要的项目介绍，引导读者去看完整的文档网站

```
---

## 3. 如何编写文档

### 3.1. 代码内文档 (Docstrings)

这是最重要、最高频的文档编写工作。当你添加或修改任何模块、类、方法或函数时，**必须** 同步更新其 `docstring`。

我们采用 **Google Python Style** 格式。

**示例 (`/src/your_package/module_a.py`):**

```python
"""
这是模块级别的 Docstring，简要介绍该文件的功能。
"""

def process_data(data: list, option: str = 'default') -> dict:
    """处理输入数据并返回结果。

    这是一个功能的详细描述。它可以跨越多行，解释该函数的核心逻辑、
    用途和注意事项。

    Args:
        data: 需要被处理的数据列表。每个元素都应该是一个整数。
        option: 一个配置选项，决定处理模式。

    Returns:
        一个包含处理结果的字典。例如：{'processed': True, 'count': 5}。

    Raises:
        ValueError: 如果 `data` 包含非整数元素。
        KeyError: 如果 `option` 无效。
    """
    if not all(isinstance(i, int) for i in data):
        raise ValueError("All elements in data must be integers.")
    # ... 函数的其余逻辑 ...
    return {'processed': True, 'count': len(data)}
```

**要点:**

- **简洁摘要行**: 第一行是功能的简短总结。
- **详细描述**: 空一行后是更详细的说明。
- **结构化字段**: 使用 `Args:`, `Returns:`, `Raises:` 等清晰地描述接口。
- **类型提示 (Type Hinting)**: 配合 Docstring，提供更严格的接口定义。



### 3.2. 项目级文档 (`/docs` 文件夹)



此文件夹中的文件使用 **Markdown (`.md`)** 格式编写，用于解释那些无法在代码 `docstring` 中说清楚的宏观概念。

- **`index.md`**: 文档网站的着陆页。应包含项目简介、核心特性和导航链接。
- **`installation.md`**: 清晰、可复现的安装步骤，包括系统依赖、Python 环境、包安装等。
- **`quickstart.md`**: 一个手把手的教程，引导新用户在 5-10 分钟内完成一个最小可用示例。
- **`architecture.md`**: (可选，但推荐) 描绘项目的主要组件、它们之间的关系以及数据流。可以使用 Mermaid.js 语法绘制图表。

## 4. 生成和预览文档

当您完成了文档的编写或修改后，可以在项目根目录运行以下命令来生成完整的 HTML 网站并本地预览：

Bash

```
# (这通常是一个封装好的命令，具体取决于项目配置)
make docs-serve
```

此命令会自动扫描 `src` 目录提取所有 `docstrings`，并与 `docs` 目录下的 Markdown 文件结合，启动一个本地 Web 服务器供您预览效果。确保在提交代码前，生成的文档是完整且无误的。

