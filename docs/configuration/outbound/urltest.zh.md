### 结构

```json
{
  "type": "urltest",
  "tag": "auto",
  
  "outbounds": [
    "proxy-a",
    "proxy-b",
    "proxy-c"
  ],
  "url": "",
  "interval": "",
  "tolerance": 50,
  "idle_timeout": "",
  "interrupt_exist_connections": false,
  "consecutive_failure_limit": 0
}
```

### 字段

#### outbounds

==必填==

用于测试的出站标签列表。

#### url

用于测试的链接。默认使用 `https://www.gstatic.com/generate_204`。

#### interval

测试间隔。 默认使用 `3m`。

#### tolerance

以毫秒为单位的测试容差。 默认使用 `50`。

#### idle_timeout

空闲超时。默认使用 `30m`。

#### interrupt_exist_connections

当选定的出站发生更改时，中断现有连接。

仅入站连接受此设置影响，内部连接将始终被中断。

#### consecutive_failure_limit

当前选定出站连续连接失败达到此次数后，自动切换到下一个最佳出站。默认值 `0` 表示禁用此功能。

当达到限制时，失败出站的延迟历史将被删除，并立即重新选择最佳可用出站，无需等待下一个健康检查周期。任何成功连接或选定出站发生变更时，失败计数器将重置。