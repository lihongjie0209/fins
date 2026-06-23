# Omron FINS TCP Protocol Driver

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue.svg)](https://golang.org/dl/)
[![Test Coverage](https://img.shields.io/badge/coverage-83.8%25-green.svg)](https://github.com/anviod/fins)

## 概述 / Overview

本项目提供了一个完整的Omron FINS TCP协议实现，用于与Omron PLC进行通信。驱动采用分层架构设计，具有良好的可扩展性、可维护性和可测试性。

This project provides a complete Omron FINS TCP protocol implementation for communicating with Omron PLCs. The driver features a layered architecture design with excellent extensibility, maintainability, and testability.

## 功能特性 / Features

- ✅ FINS TCP协议帧编解码 (FINS TCP frame encoding/decoding)
- ✅ TCP连接管理与自动重连 (TCP connection management with auto-reconnection)
- ✅ 心跳检测机制 (Heartbeat monitoring)
- ✅ 批量点位优化调度 (Batch point optimization)
- ✅ 支持多种数据类型 (Multiple data types support)
- ✅ 支持多种内存区域 (Multiple memory areas support)
- ✅ 分级错误处理 (Hierarchical error handling)
- ✅ 线程安全设计 (Thread-safe design)
- ✅ Mock PLC服务器用于测试 (Mock PLC server for testing)

## 支持的内存区域 / Supported Memory Areas

| 区域 / Area | 描述 / Description | 读写权限 / Read/Write |
|-------------|--------------------|----------------------|
| CIO | Input/Output relays | Read/Write |
| W | Work relays | Read/Write |
| H | Holding relays | Read/Write |
| A | Auxiliary relays | Read-only |
| D | Data Memory | Read/Write |
| P | PVs | Read/Write |
| F | Flag area | Read-only |
| EM0-EM15 | Extended Memory | Read/Write |
| T/C | Timer/Counter | Read/Write |

## 支持的数据类型 / Supported Data Types

| 数据类型 / Type | 说明 / Description | 字节数 / Size |
|-----------------|--------------------|---------------|
| BIT | Single bit | 1 bit |
| UINT8 | Unsigned 8-bit integer | 1 byte |
| INT8 | Signed 8-bit integer | 1 byte |
| UINT16 | Unsigned 16-bit integer | 2 bytes |
| INT16 | Signed 16-bit integer | 2 bytes |
| UINT32 | Unsigned 32-bit integer | 4 bytes |
| INT32 | Signed 32-bit integer | 4 bytes |
| UINT64 | Unsigned 64-bit integer | 8 bytes |
| INT64 | Signed 64-bit integer | 8 bytes |
| FLOAT | 32-bit floating point | 4 bytes |
| DOUBLE | 64-bit floating point | 8 bytes |
| STRING | String data | configurable |

## 地址格式 / Address Format

地址字符串格式: `AREA ADDRESS[.BIT][.LEN[H][L]]`

Address string format: `AREA ADDRESS[.BIT][.LEN[H][L]]`

| 示例 / Example | 说明 / Description |
|----------------|--------------------|
| `D100` | Data Memory area, word address 100 |
| `CIO0.0` | CIO area, address 0, bit 0 |
| `D10.20` | D area, address 10, string length 20 (byte order L) |
| `D10.20H` | D area, address 10, string length 20, byte order H |
| `EM10W100` | EM10 area, word address 100 |
| `EM10W100.5` | EM10 area, address 100, bit 5 |
| `F0` | Flag area, bit address 0 |

## 快速开始 / Quick Start

### 安装 / Installation

```bash
go get github.com/anviod/fins
```

### 基本使用 / Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/anviod/fins"
)

func main() {
    // 创建驱动 / Create driver
    driver := fins.NewFinsTCPDriver()

    // 配置驱动 / Configure driver
    cfg := fins.DriverConfig{
        Protocol: "omron-fins-tcp",
        Config: map[string]interface{}{
            "plcIP":   "192.168.1.100",
            "plcPort": 9600,
            "timeout": 3000,
        },
    }

    // 初始化驱动 / Initialize driver
    if err := driver.Init(cfg); err != nil {
        panic(err)
    }

    // 建立连接 / Connect
    ctx := context.Background()
    if err := driver.Connect(ctx); err != nil {
        panic(err)
    }
    defer driver.Disconnect()

    // 读取点位 / Read points
    points := []fins.Point{
        {ID: "temp", Address: "D100", DataType: fins.DataTypeFLOAT},
        {ID: "status", Address: "CIO0.0", DataType: fins.DataTypeBIT},
    }

    results, err := driver.ReadPoints(ctx, points)
    if err != nil {
        panic(err)
    }

    // 输出结果 / Print results
    for id, val := range results {
        fmt.Printf("%s: %v (quality: %s)\n", id, val.Value, val.Quality)
    }
}
```

### 写入操作 / Write Operation

```go
// 写入UINT16值
point := fins.Point{ID: "output", Address: "D200", DataType: fins.DataTypeUINT16}
err := driver.WritePoint(ctx, point, uint16(1234))

// 写入浮点值
point2 := fins.Point{ID: "setpoint", Address: "D300", DataType: fins.DataTypeFLOAT}
err = driver.WritePoint(ctx, point2, float32(25.5))

// 写入位值
point3 := fins.Point{ID: "coil", Address: "CIO100.0", DataType: fins.DataTypeBIT}
err = driver.WritePoint(ctx, point3, true)
```

### 健康监控 / Health Monitoring

```go
status := driver.Health()
metrics := driver.GetConnectionMetrics()

fmt.Printf("Health: %s, Connected: %v, Reconnects: %d\n",
    status, metrics.Connected, metrics.TotalReconnects)
```

## 配置参数 / Configuration Parameters

| 参数 / Parameter | 类型 / Type | 默认值 / Default | 说明 / Description |
|------------------|-------------|------------------|--------------------|
| plcIP | string | - | PLC IPv4地址 (required) |
| plcPort | int | 9600 | PLC端口号 |
| timeout | int | 3000 | 通信超时(毫秒) |
| maxFrameLength | int | 64 | 最大读取字数 |
| heartbeatInterval | int | 30000 | 心跳间隔(毫秒) |
| maxRetries | int | 3 | 最大重连次数 |
| retryInterval | int | 1000 | 重连基础间隔(毫秒) |
| srcNetworkAddr | int | 0 | 源网络地址 |
| srcNodeAddr | int | 1 | 源节点地址 |
| srcUnitAddr | int | 255 | 源单元地址 |
| dstNetworkAddr | int | 0 | 目标网络地址 |
| dstNodeAddr | int | 1 | 目标节点地址 |
| dstUnitAddr | int | 0 | 目标单元地址 |

## 项目结构 / Project Structure

```
fins/
├── protocol.go        # 协议层 - FINS TCP帧编解码
├── transport.go       # 传输层 - TCP连接管理
├── decoder.go         # 解码器 - 地址解析与数据编解码
├── scheduler.go       # 调度器 - 批量点位优化
├── fins.go            # 驱动主模块 - Driver接口实现
├── types.go           # 公共类型定义
├── mock_plc.go        # Mock PLC服务器
├── doc.go             # 包级文档
├── udp/               # UDP FINS协议实现(旧版)
└── *test.go           # 测试文件
```

## 分层架构 / Layered Architecture

```
┌──────────────────────────────────────────────┐
│              Driver Layer                    │
│        (fins.go - FinsTCPDriver)             │
│   标准Driver接口，模块协调，对外API           │
├──────────────────────────────────────────────┤
│            Scheduler Layer                   │
│        (scheduler.go - Scheduler)            │
│   批量点位分组，读写调度，统计指标           │
├──────────────────────────────────────────────┤
│             Decoder Layer                    │
│         (decoder.go - Decoder)               │
│   地址解析，数据类型编解码，区域映射         │
├──────────────────────────────────────────────┤
│            Transport Layer                   │
│        (transport.go - Transport)            │
│   TCP连接管理，心跳检测，自动重连           │
├──────────────────────────────────────────────┤
│            Protocol Layer                    │
│         (protocol.go - 协议常量/编解码)      │
│   FINS TCP帧结构，命令码，结束码           │
└──────────────────────────────────────────────┘
```

## 测试 / Testing

运行所有测试:

Run all tests:

```bash
go test -v -cover
```

运行特定模块测试:

Run specific module tests:

```bash
go test -v -run TestTransport
go test -v -run TestDecoder
go test -v -run TestScheduler
```

测试覆盖率: **83.8%**

## 错误处理 / Error Handling

驱动实现了分级错误处理策略:

The driver implements hierarchical error handling:

1. **连接错误** - 自动重连，指数退避策略
   Connection errors - Auto-reconnection with exponential backoff

2. **协议错误** - 直接返回给调用方
   Protocol errors - Returned directly to caller

3. **数据错误** - 标记为Bad质量
   Data errors - Marked with Bad quality

4. **超时错误** - 可配置超时和重试
   Timeout errors - Configurable timeout with retry

获取错误描述:

Get error description:

```go
code := fins.END_CODE_ADDR_RANGE_EXCEED
fmt.Println(fins.EndCodeText(code)) // "address range exceeded"
```

## 调试 / Debugging

启用调试日志:

Enable debug logging:

```bash
export FINS_DEBUG=1
go run your_program.go
```

## 参考文档 / References

- Omron FINS Command Reference: W342-E1-15
- CP Series Communication: W343
- NJ/NX Series FINS: W527

## 许可证 / License

MIT License

## 贡献 / Contributing

欢迎提交Issue和Pull Request！

Contributions are welcome!

---

**Omron** 和 **FINS** 是欧姆龙公司的商标。

**Omron** and **FINS** are trademarks of Omron Corporation.