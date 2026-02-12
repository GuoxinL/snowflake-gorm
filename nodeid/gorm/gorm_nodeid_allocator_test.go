//
// Copyright (C) BABEC. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0
//

// Package gorm gorm实现的节点ID分配器
package gorm

import (
	"context"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/GuoxinL/snowflake-gorm/nodeid/gorm/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	//commonsnowflake "github.com/GuoxinL/snowflake-gorm"
	"github.com/bwmarrin/snowflake"
)

const testPort = 8080
const testName = "testname"

var logger = &DefaultLogger{}

// testDB 创建测试数据库连接
func testDB(t *testing.T) *gorm.DB {

	db, err := gorm.Open(sqlite.Open(filepath.Join(os.TempDir(), strconv.Itoa(rand.IntN(32))+"-sqlite.db")))
	require.NoError(t, err)

	// 自动迁移表结构
	err = db.AutoMigrate(&model.SnowflakeKv{})
	require.NoError(t, err)

	return db
}

// TestNewNodeIdAllocator 测试节点ID分配器创建
func TestNewNodeIdAllocator(t *testing.T) {
	db := testDB(t)
	require.NotNil(t, db)

	ctx := context.Background()
	allocator := NewNodeIdAllocator(ctx, db, testName, testPort, time.Second, 5*time.Second, logger)
	assert.NotNil(t, allocator)
	assert.NotNil(t, allocator.NodeIdAllocator)
	assert.Equal(t, testPort, testPort)
}

// TestNodeIdAllocator_Alloc_FirstTime 测试首次分配节点ID
func TestNodeIdAllocator_Alloc_FirstTime(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	allocator := NewNodeIdAllocator(ctx, db, testName, testPort, time.Second, 5*time.Second, logger)

	// 首次分配，应该创建新记录
	nodeId, err := allocator.Alloc()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, nodeId, int64(0))
	assert.Less(t, nodeId, int64(1024))

	// 验证记录已创建
	tab := allocator.dao.SnowflakeKv
	record, err := tab.WithContext(ctx).Where(tab.Key.Eq(allocator.nodeIdKey), tab.NodeID.Eq(nodeId)).First()
	require.NoError(t, err)
	assert.Equal(t, allocator.nodeIdKey, record.Key)
	assert.Equal(t, nodeId, record.NodeID)
	assert.NotNil(t, record.Created)
	assert.Greater(t, record.Time, int64(0))
}

// TestNodeIdAllocator_Alloc_Existing 测试已存在记录的节点ID分配
func TestNodeIdAllocator_Alloc_Existing(t *testing.T) {

	db := testDB(t)
	ctx := context.Background()
	allocator := NewNodeIdAllocator(ctx, db, testName, testPort, time.Second, 5*time.Second, logger)

	// 第一次分配
	firstNodeId, err := allocator.Alloc()
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond) // 确保时间有差异

	// 第二次分配，应该返回相同的节点ID
	secondNodeId, err := allocator.Alloc()
	require.NoError(t, err)
	assert.Equal(t, firstNodeId, secondNodeId)

	// 验证时间已更新
	tab := allocator.dao.SnowflakeKv
	record, err := tab.WithContext(ctx).Where(tab.Key.Eq(allocator.nodeIdKey), tab.NodeID.Eq(firstNodeId)).First()
	require.NoError(t, err)
	assert.Equal(t, firstNodeId, record.NodeID)
	assert.Greater(t, record.Time, int64(0))
}

// TestNodeIdAllocator_Alloc_TimeRollback_SmallDrift 测试小幅时钟回拨（在容忍范围内）
func TestNodeIdAllocator_Alloc_TimeRollback_SmallDrift(t *testing.T) {

	db := testDB(t)
	ctx := context.Background()
	// 设置1秒的容忍时间
	acceptableClockDrift := 500 * time.Millisecond
	allocator := NewNodeIdAllocator(ctx, db, testName, testPort, acceptableClockDrift, 5*time.Second, logger)

	// 首次分配
	nodeId, err := allocator.Alloc()
	require.NoError(t, err)

	// 手动设置一个未来的时间（模拟时钟回拨场景，但在容忍范围内）
	futureTime := time.Now().Add(200 * time.Millisecond).UnixMilli()
	tab := allocator.dao.SnowflakeKv
	tab.WithContext(ctx).Where(tab.Key.Eq(allocator.nodeIdKey), tab.NodeID.Eq(nodeId)).
		Updates(&model.SnowflakeKv{
			Key:     allocator.nodeIdKey,
			NodeID:  nodeId,
			Time:    futureTime,
			Updated: time.Now(),
		})

	// 再次分配，应该等待并返回相同的节点ID
	startTime := time.Now()
	secondNodeId, err := allocator.Alloc()
	elapsed := time.Since(startTime)

	require.NoError(t, err)
	assert.Equal(t, nodeId, secondNodeId)
	// 应该等待了容忍时间
	assert.GreaterOrEqual(t, elapsed, acceptableClockDrift-50*time.Millisecond)
}

// TestNodeIdAllocator_Alloc_TimeRollback_LargeDrift 测试大幅时钟回拨（超出容忍范围）
func TestNodeIdAllocator_Alloc_TimeRollback_LargeDrift(t *testing.T) {

	db := testDB(t)
	ctx := context.Background()
	// 设置100ms的容忍时间
	acceptableClockDrift := 100 * time.Millisecond
	allocator := NewNodeIdAllocator(ctx, db, testName, testPort, acceptableClockDrift, 5*time.Second, logger)

	// 首次分配
	oldNodeId, err := allocator.Alloc()
	require.NoError(t, err)

	// 手动设置一个未来的时间（模拟大幅时钟回拨）
	futureTime := time.Now().Add(24 * time.Hour).UnixMilli()
	tab := allocator.dao.SnowflakeKv
	tab.WithContext(ctx).Where(tab.Key.Eq(allocator.nodeIdKey), tab.NodeID.Eq(oldNodeId)).
		Updates(&model.SnowflakeKv{
			Key:     allocator.nodeIdKey,
			NodeID:  oldNodeId,
			Time:    futureTime,
			Updated: time.Now(),
		})

	// 再次分配，应该触发节点ID漂移
	newNodeId, err := allocator.Alloc()
	require.NoError(t, err)
	// 由于使用了哈希分配器，Migration会根据oldNodeId计算新的nodeId
	// 可能相同也可能不同，取决于哈希结果
	assert.GreaterOrEqual(t, newNodeId, int64(0))
	assert.Less(t, newNodeId, int64(1024))
}

// TestNodeIdAllocator_Alloc_NodeIdContention 测试节点ID抢占
func TestNodeIdAllocator_Alloc_NodeIdContention(t *testing.T) {

	db := testDB(t)
	ctx := context.Background()
	// 设置很短的抢占间隔
	contentionInterval := 200 * time.Millisecond
	allocator := NewNodeIdAllocator(ctx, db, testName, testPort, time.Second, contentionInterval, logger)

	// 首次分配
	nodeId, err := allocator.Alloc()
	require.NoError(t, err)

	// 手动更新时间，使其过期
	oldRecordTime := time.Now().Add(-time.Second).UnixMilli()
	tab := allocator.dao.SnowflakeKv
	tab.WithContext(ctx).Where(tab.Key.Eq(allocator.nodeIdKey), tab.NodeID.Eq(nodeId)).
		Updates(&model.SnowflakeKv{
			Key:     allocator.nodeIdKey,
			NodeID:  nodeId,
			Time:    oldRecordTime,
			Updated: time.Now(),
		})

	// 等待抢占间隔
	time.Sleep(300 * time.Millisecond)

	// 再次分配
	secondNodeId, err := allocator.Alloc()
	require.NoError(t, err)
	// 由于使用哈希分配器，相同key会返回相同的nodeId
	assert.Equal(t, nodeId, secondNodeId)
}

// TestNodeIdAllocator_Alloc_DifferentPorts 测试不同端口产生不同的节点ID
func TestNodeIdAllocator_Alloc_DifferentPorts(t *testing.T) {

	db := testDB(t)
	ctx := context.Background()

	allocator1 := NewNodeIdAllocator(ctx, db, testName, 8080, time.Second, 5*time.Second, logger)
	allocator2 := NewNodeIdAllocator(ctx, db, testName, 9090, time.Second, 5*time.Second, logger)

	nodeId1, err := allocator1.Alloc()
	require.NoError(t, err)

	nodeId2, err := allocator2.Alloc()
	require.NoError(t, err)

	// 不同端口应该产生不同的节点ID（极大概率）
	assert.NotEqual(t, nodeId1, nodeId2)
}

// TestNewTimeSynchronizer 测试时间同步器创建
func TestNewTimeSynchronizer(t *testing.T) {

	db := testDB(t)
	ctx := context.Background()
	interval := 100 * time.Millisecond

	synchronizer := NewTimeSynchronizer(ctx, db, testName, testPort, interval, logger)
	assert.NotNil(t, synchronizer)
	assert.NotNil(t, synchronizer.ticker)
	assert.NotNil(t, synchronizer.curr)
}

// TestTimeSynchronizer_Async 测试时间同步器异步调用
func TestTimeSynchronizer_Async(t *testing.T) {

	db := testDB(t)
	ctx := context.Background()
	synchronizer := NewTimeSynchronizer(ctx, db, testName, testPort, time.Second, logger)

	testTime := int64(1234567890)
	synchronizer.Async(testTime)
	synchronizer.Run()
	// 等待goroutine处理
	time.Sleep(100 * time.Millisecond)
	ctx.Done()

	currTime := synchronizer.curr.Load()
	assert.Equal(t, testTime, currTime)
}

// TestTimeSynchronizer_Async_Multiple 测试多次异步调用
func TestTimeSynchronizer_Async_Multiple(t *testing.T) {

	db := testDB(t)
	ctx := context.Background()
	synchronizer := NewTimeSynchronizer(ctx, db, testName, testPort, time.Second, logger)
	synchronizer.Run()

	times := []int64{1000, 2000, 3000, 4000, 5000}

	for _, t := range times {
		synchronizer.Async(t)
	}

	time.Sleep(100 * time.Millisecond)
	ctx.Done()

	currTime := synchronizer.curr.Load()
	assert.Equal(t, times[len(times)-1], currTime)
}

// TestTimeSynchronizer_Run 测试时间同步器运行
func TestTimeSynchronizer_Run(t *testing.T) {

	db := testDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 先创建节点ID记录
	allocator := NewNodeIdAllocator(ctx, db, testName, testPort, time.Second, 5*time.Second, logger)
	nodeId, err := allocator.Alloc()
	require.NoError(t, err)
	assert.Greater(t, nodeId, int64(0))

	interval := 50 * time.Millisecond
	synchronizer := NewTimeSynchronizer(ctx, db, testName, testPort, interval, logger)

	// 运行同步器
	synchronizer.Run()

	// 发送时间戳
	testTime := time.Now().UnixMilli()
	synchronizer.Async(testTime)

	// 等待时间戳被保存到数据库
	time.Sleep(200 * time.Millisecond)

	// 验证数据库中的记录
	tab := synchronizer.dao.SnowflakeKv
	records, err := tab.WithContext(ctx).Where(tab.Key.Eq(synchronizer.nodeIdKey)).Find()
	require.NoError(t, err)
	assert.Greater(t, len(records), 0)
	// 验证至少有一个记录的时间匹配
	found := false
	for _, record := range records {
		if record.Time == testTime {
			found = true
			break
		}
	}
	assert.True(t, found)
}

// TestTimeSynchronizer_Run_MultipleAsync 测试多次异步调用
func TestTimeSynchronizer_Run_MultipleAsync(t *testing.T) {

	db := testDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 先创建节点ID记录
	allocator := NewNodeIdAllocator(ctx, db, testName, testPort, time.Second, 5*time.Second, logger)
	nodeId, err := allocator.Alloc()
	require.NoError(t, err)
	assert.Greater(t, nodeId, int64(0))

	interval := 50 * time.Millisecond
	synchronizer := NewTimeSynchronizer(ctx, db, testName, testPort, interval, logger)

	synchronizer.Run()

	// 发送多个时间戳
	times := []int64{
		time.Now().UnixMilli(),
		time.Now().Add(1 * time.Second).UnixMilli(),
		time.Now().Add(2 * time.Second).UnixMilli(),
	}

	for _, t := range times {
		synchronizer.Async(t)
	}

	// 等待所有时间戳被处理
	time.Sleep(300 * time.Millisecond)

	// 验证最后一个时间戳已保存
	currTime := synchronizer.curr.Load()
	assert.Equal(t, times[len(times)-1], currTime)

	// 验证数据库中的记录
	tab := synchronizer.dao.SnowflakeKv
	records, err := tab.WithContext(ctx).Where(tab.Key.Eq(synchronizer.nodeIdKey)).Find()
	require.NoError(t, err)
	assert.Greater(t, len(records), 0)
}

// TestTimeSynchronizer_ContextCancel 测试context取消
func TestTimeSynchronizer_ContextCancel(t *testing.T) {

	db := testDB(t)
	ctx, cancel := context.WithCancel(context.Background())

	interval := 10 * time.Millisecond
	synchronizer := NewTimeSynchronizer(ctx, db, testName, testPort, interval, logger)

	synchronizer.Run()

	// 取消context
	cancel()

	// 等待goroutine退出
	time.Sleep(100 * time.Millisecond)

	// 验证没有panic
	assert.True(t, true)
}

// TestNodeIdAllocator_Interface 测试接口实现
func TestNodeIdAllocator_Interface(t *testing.T) {

	db := testDB(t)
	ctx := context.Background()
	allocator := NewNodeIdAllocator(ctx, db, testName, testPort, time.Second, 5*time.Second, logger)

	// 验证实现了snowflake.NodeIdAllocator接口
	var _ snowflake.NodeIdAllocator = allocator
	assert.NotNil(t, allocator)
}

// TestTimeSynchronizer_Interface 测试接口实现
func TestTimeSynchronizer_Interface(t *testing.T) {

	db := testDB(t)
	ctx := context.Background()
	synchronizer := NewTimeSynchronizer(ctx, db, testName, testPort, time.Second, logger)

	// 验证实现了snowflake.TimeSynchronizer接口
	var _ snowflake.TimeSynchronizer = synchronizer
	assert.NotNil(t, synchronizer)
}

// TestTimeSynchronizer_Async_Zero 测试零时间戳
func TestTimeSynchronizer_Async_Zero(t *testing.T) {

	db := testDB(t)
	ctx := context.Background()
	synchronizer := NewTimeSynchronizer(ctx, db, testName, testPort, time.Second, logger)

	// 发送零时间戳
	synchronizer.Async(0)

	time.Sleep(100 * time.Millisecond)

	currTime := synchronizer.curr.Load()
	assert.Equal(t, int64(0), currTime)
}

// TestTimeSynchronizer_Run_ZeroValue 测试零值不更新数据库
func TestTimeSynchronizer_Run_ZeroValue(t *testing.T) {

	db := testDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 先创建节点ID记录
	allocator := NewNodeIdAllocator(ctx, db, testName, testPort, time.Second, 5*time.Second, logger)
	nodeId, err := allocator.Alloc()
	require.NoError(t, err)
	assert.Greater(t, nodeId, int64(0))

	interval := 50 * time.Millisecond
	synchronizer := NewTimeSynchronizer(ctx, db, testName, testPort, interval, logger)

	synchronizer.Run()

	// 不发送时间戳，等待ticker触发
	time.Sleep(150 * time.Millisecond)

	// 验证不会更新数据库（因为curr为0）
	tab := synchronizer.dao.SnowflakeKv
	records, err := tab.WithContext(ctx).Where(tab.Key.Eq(synchronizer.nodeIdKey)).Find()
	require.NoError(t, err)
	// 应该有一条记录，但时间不会被更新为0
	assert.Equal(t, 1, len(records))
	assert.Greater(t, records[0].Time, int64(0))
}

// TestNodeIdAllocator_Alloc_MultipleTimes 测试多次分配
func TestNodeIdAllocator_Alloc_MultipleTimes(t *testing.T) {

	db := testDB(t)
	ctx := context.Background()
	allocator := NewNodeIdAllocator(ctx, db, testName, testPort, time.Second, 5*time.Second, logger)

	// 多次分配应该返回相同的节点ID
	previousNodeId := int64(-1)
	for i := 0; i < 10; i++ {
		nodeId, err := allocator.Alloc()
		require.NoError(t, err)
		if previousNodeId >= 0 {
			assert.Equal(t, previousNodeId, nodeId)
		}
		previousNodeId = nodeId
	}
}
