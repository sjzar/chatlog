Multi文件夹中的文件解码和之前的其它数据库操作相同。

该文件夹中文件结构比较简单，只有三种：FTSMSG、MediaMSG和MSG。这里说是三种不是三个，是因为这里的数据库达到一定大小后会拆分。

FTSMSG
看过《总述》一文的应该很熟悉 FTS 这一前缀了——这代表的是搜索时所需的索引。

其内主要的内容是这样的两张表：

FTSChatMsg2_content：内有三个字段
docid：从1开始递增的数字，相当于当前条目的 ID
c0content：搜索关键字（在微信搜索框输入的关键字被这个字段包含的内容可以被搜索到）
c1entityId：尚不明确用途，可能是校验相关
FTSChatMsg2_MetaData
docid：与FTSChatMsg2_content表中的 docid 对应
msgId：与MSG数据库中的内容对应
entityId：与FTSChatMsg2_content表中的 c1entityId 对应
type：可能是该消息的类型
其余字段尚不明确
特别地，表名中的这个数字2，个人猜测可能是当前数据库格式的版本号。

MediaMSG
这里存储了所有的语音消息。数据库中有且仅有Media一张表，内含三个有效字段：

Key
Reserved0
Buf
其中Reserved0字段与MSG数据库中消息的MsgSvrID一一对应。

第三项即语音的二进制数据，观察头部即可发现这些文件是以 SILK 格式存储的。这是一种微软为 Skype 开发并开源的语音格式，具体可以自行 Google。

下面是将 Buf 字段中的数据导出为文件的代码：

import sqlite3


def writeTofile(data, filename):
    with open(filename, 'wb') as file:
        file.write(data)
    print("Stored blob data into: ", filename, "\n")

def readBlobData(key):
    try:
        sqliteConnection = sqlite3.connect('dbs/decoded_MediaMSG0.db')
        cursor = sqliteConnection.cursor(<