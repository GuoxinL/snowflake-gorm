//
// Copyright (C) BABEC. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0
//

// Package snowflake
package snowflake

import (
	"context"
	"time"

	nodeidgorm "github.com/GuoxinL/snowflake-gorm/nodeid/gorm"
	"github.com/bwmarrin/snowflake"
	"gorm.io/gorm"
)

type Config struct {
	DB                       *gorm.DB
	Port                     int
	AcceptableClockDrift     time.Duration
	NodeIdContentionInterval time.Duration
}

// NewSnowflake 创建一个雪花算法
// @param config
// @return *snowflake.Node
// @return error
func NewSnowflake(ctx context.Context, db *gorm.DB, name string, port int, acceptableClockDrift,
	nodeIdContentionInterval time.Duration, logger nodeidgorm.Logger) (*snowflake.Node, error) {
	// 1. 节点id分配器
	allocator := nodeidgorm.NewNodeIdAllocator(ctx, db, name, port, acceptableClockDrift, nodeIdContentionInterval, logger)
	// 2. 时间同步器
	synchronizer := nodeidgorm.NewTimeSynchronizer(ctx, db, name, port, acceptableClockDrift, logger)
	// 2.1 启动时间同步器
	synchronizer.Run()
	// 3. 雪花算法
	option, err := snowflake.NewWithOption(snowflake.WithNodeIdAllocator(allocator), snowflake.WithTimeSynchronizer(synchronizer))
	if err != nil {
		return nil, err
	}
	return option, nil
}
