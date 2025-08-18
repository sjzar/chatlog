# 语音消息调试指南

## 概述

本指南说明了如何使用新增的详细日志功能来调试语音消息获取过程中的问题。我们在语音处理的每个关键层面都添加了详细的日志记录。

## 日志层级

### 1. HTTP路由层 (`internal/chatlog/http/route.go`)
记录的信息：
- 请求参数（语音key）
- 参数验证结果
- 媒体文件获取过程
- 语音数据处理结果
- SILK到MP3转换状态

### 2. 数据库抽象层 (`internal/wechatdb/wechatdb.go`)
记录的信息：
- 媒体文件获取请求
- 平台和版本信息
- 处理结果和数据大小

### 3. Repository层 (`internal/wechatdb/repository/media.go`)
记录的信息：
- Repository层的处理过程
- 数据源调用结果

### 4. 数据源层 (各版本datasource实现)
记录的信息：
- 数据库连接获取
- SQL查询执行
- 查询结果扫描
- 数据库搜索过程

### 5. 数据库管理器层 (`internal/wechatdb/datasource/dbm/dbm.go`)
记录的信息：
- 数据库组连接过程
- 文件列表获取
- 数据库连接状态

### 6. SILK转换层 (`pkg/util/silk/silk.go`)
记录的信息：
- SILK解码过程
- PCM数据大小
- MP3编码过程
- 转换成功/失败状态

## 日志级别配置

### Debug级别
包含最详细的信息，适合深度调试：
```bash
# 设置环境变量启用Debug日志
export LOG_LEVEL=debug
./chatlog server
```

### Info级别
包含关键步骤信息：
```bash
export LOG_LEVEL=info
./chatlog server
```

### Error级别
仅显示错误信息：
```bash
export LOG_LEVEL=error
./chatlog server
```

## 常见问题调试

### 1. "no such table: Media" 错误

查看日志输出中的关键信息：
```
level=error msg="执行语音查询失败" key="xxxxx" db_index=0 query="SELECT Buf FROM Media WHERE Reserved0 = ?" error="no such table: Media"
```

这表明：
- 使用的是Windows V3数据源
- 数据库中没有Media表
- 可能是版本识别错误

**解决方案：**
1. 检查数据库文件是否完整
2. 确认微信版本识别是否正确
3. 查看数据库连接日志确认连接了正确的文件

### 2. 语音数据为空

查看日志中的数据大小：
```
level=warn msg="语音数据为空，继续查找" key="xxxxx" db_index=0 row_index=1
```

这表明：
- 找到了数据库记录
- 但语音数据字段为空

**解决方案：**
1. 检查语音消息是否已经被清理
2. 确认key是否正确
3. 尝试其他数据库文件

### 3. 数据库连接失败

查看数据库管理器日志：
```
level=error msg="连接数据库文件失败" group_name="voice" file_index=0 file_path="/path/to/MediaMSG.db"
```

**解决方案：**
1. 确认文件路径正确
2. 检查文件权限
3. 确认微信进程已关闭

## 日志示例

### 成功的语音获取流程：
```
level=info msg="开始处理语音消息请求" key="1234567890"
level=info msg="开始获取媒体文件" type="voice" key="1234567890"
level=info msg="开始获取媒体文件" type="voice" key="1234567890" platform="windows" version=3
level=debug msg="Repository层开始获取媒体文件" type="voice" key="1234567890"
level=info msg="开始获取语音数据 (Windows V3)" key="1234567890"
level=debug msg="开始获取数据库连接组" group_name="voice"
level=info msg="成功获取数据库连接组" group_name="voice" connected_db_count=2 total_files=2
level=debug msg="找到语音数据行" key="1234567890" db_index=0 row_index=1
level=info msg="成功获取语音数据 (Windows V3)" key="1234567890" db_index=0 voice_data_size=8192
level=info msg="成功获取媒体文件" type="voice" key="1234567890" platform="windows" version=3 data_size=8192
level=info msg="开始处理语音数据转换" input_data_size=8192
level=info msg="开始SILK到MP3转换" input_data_size=8192
level=info msg="SILK解码成功" silk_data_size=8192 pcm_data_size=49152
level=info msg="MP3编码成功" pcm_data_size=49152 mp3_data_size=3072
level=info msg="SILK到MP3转换成功" input_size=8192 output_size=3072
```

## 性能注意事项

- Debug级别日志会影响性能，生产环境建议使用Info级别
- 日志文件可能会快速增长，注意配置日志轮转
- 语音数据大小日志可以帮助识别异常大小的文件

## 自定义日志输出

如果需要将日志输出到文件：
```bash
./chatlog server 2>&1 | tee voice_debug.log
```

或者只保存错误日志：
```bash
./chatlog server 2>voice_errors.log
``` 