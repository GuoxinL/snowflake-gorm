//
// Copyright (C) BABEC. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0
//

// Package snowflake benchmark
package snowflake

import (
	"context"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	nodeidgorm "github.com/GuoxinL/snowflake-gorm/nodeid/gorm"
	"github.com/GuoxinL/snowflake-gorm/nodeid/gorm/model"
	"github.com/bwmarrin/snowflake"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var logger = &nodeidgorm.DefaultLogger{}

// setupTestDB 创建测试数据库连接并初始化表结构
func setupTestDB(t testing.TB) *gorm.DB {

	db, err := gorm.Open(sqlite.Open(filepath.Join(os.TempDir(), strconv.Itoa(rand.IntN(32))+"-sqlite.db")))
	require.NoError(t, err)

	// 自动迁移表结构
	err = db.AutoMigrate(&model.SnowflakeKv{})
	require.NoError(t, err)

	return db
}

// BenchmarkNewSnowflake_Creation 测试创建雪花算法实例的性能
func BenchmarkNewSnowflake_Creation(b *testing.B) {
	ctx := context.Background()
	db := setupTestDB(b)

	port := 8080
	name := "test_name"
	acceptableClockDrift := time.Second
	nodeIdContentionInterval := 5 * time.Second

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 每次创建一个新的 context，避免 ticker 泄漏
		testCtx, cancel := context.WithCancel(ctx)
		sf, err := NewSnowflake(testCtx, db, name, port, acceptableClockDrift, nodeIdContentionInterval, logger)
		if err != nil {
			b.Fatal(err)
		}
		// 立即取消以停止后台 goroutine
		cancel()
		_ = sf
	}
}

// BenchmarkNewSnowflake_GenerateID 测试生成雪花ID的性能
func BenchmarkNewSnowflake_GenerateID(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := setupTestDB(b)

	port := 8080
	name := "test_name"
	acceptableClockDrift := time.Second
	nodeIdContentionInterval := 5 * time.Second

	sf, err := NewSnowflake(ctx, db, name, port, acceptableClockDrift, nodeIdContentionInterval, logger)
	require.NoError(b, err)
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sf.Generate()
	}
}

// BenchmarkGenerateMaxSequence 原生雪花算法生成最大序列号的性能测试
func BenchmarkGenerateMaxSequence(b *testing.B) {

	//snowflake.NodeBits = 1
	//snowflake.StepBits = 21
	node, _ := snowflake.NewNode(1)

	b.ReportAllocs()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = node.Generate()
	}
}

// BenchmarkNewSnowflake_GenerateID_Parallel 测试并发生成雪花ID的性能
func BenchmarkNewSnowflake_GenerateID_Parallel(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := setupTestDB(b)

	port := 8080
	name := "test_name"
	acceptableClockDrift := time.Second
	nodeIdContentionInterval := 5 * time.Second

	sf, err := NewSnowflake(ctx, db, name, port, acceptableClockDrift, nodeIdContentionInterval, logger)
	require.NoError(b, err)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = sf.Generate()
		}
	})
}

// BenchmarkNewSnowflake_MultipleInstances 测试创建多个实例的性能
func BenchmarkNewSnowflake_MultipleInstances(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := setupTestDB(b)
	name := "test_name"
	acceptableClockDrift := time.Second
	nodeIdContentionInterval := 5 * time.Second

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		port := 8080 + (i % 100) // 使用不同端口
		testCtx, ctxCancel := context.WithCancel(ctx)
		sf, err := NewSnowflake(testCtx, db, name, port, acceptableClockDrift, nodeIdContentionInterval, logger)
		if err != nil {
			b.Fatal(err)
		}
		_ = sf.Generate()
		ctxCancel()
	}
}

// BenchmarkNewSnowflake_HighThroughput 测试高吞吐量场景下的性能
func BenchmarkNewSnowflake_HighThroughput(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := setupTestDB(b)

	port := 8080
	name := "test_name"
	acceptableClockDrift := 10 * time.Millisecond // 减少容忍时间以减少等待
	nodeIdContentionInterval := 5 * time.Second

	sf, err := NewSnowflake(ctx, db, name, port, acceptableClockDrift, nodeIdContentionInterval, logger)
	require.NoError(b, err)

	var totalIDs int64
	var wg sync.WaitGroup
	workers := 10

	b.ResetTimer()

	// 启动多个 goroutine 并发生成ID
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < b.N/workers; i++ {
				_ = sf.Generate()
				atomic.AddInt64(&totalIDs, 1)
			}
		}()
	}

	wg.Wait()
	b.ReportMetric(float64(totalIDs)/b.Elapsed().Seconds(), "ids/sec")
}

// BenchmarkNewSnowflake_IDUniqueness 验证ID唯一性的性能测试
func BenchmarkNewSnowflake_IDUniqueness(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := setupTestDB(b)

	port := 8080
	name := "test_name"
	acceptableClockDrift := time.Second
	nodeIdContentionInterval := 5 * time.Second

	sf, err := NewSnowflake(ctx, db, name, port, acceptableClockDrift, nodeIdContentionInterval, logger)
	require.NoError(b, err)

	idMap := make(map[int64]struct{}, b.N)
	var mu sync.Mutex

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := sf.Generate()
		mu.Lock()
		if _, exists := idMap[id.Int64()]; exists {
			mu.Unlock()
			b.Fatalf("发现重复ID: %d", id)
		}
		idMap[id.Int64()] = struct{}{}
		mu.Unlock()
	}
}

// BenchmarkNewSnowflake_Serialization 测试序列化性能
func BenchmarkNewSnowflake_Serialization(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := setupTestDB(b)

	port := 8080
	name := "test_name"
	acceptableClockDrift := time.Second
	nodeIdContentionInterval := 5 * time.Second

	sf, err := NewSnowflake(ctx, db, name, port, acceptableClockDrift, nodeIdContentionInterval, logger)
	require.NoError(b, err)

	id := sf.Generate()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = id.String()
	}
}
