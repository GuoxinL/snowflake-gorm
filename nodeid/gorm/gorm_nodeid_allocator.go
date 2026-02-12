//
// Copyright (C) BABEC. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0
//

// Package grom gorm节点id分配器
package gorm

import (
	"context"
	"errors"
	"time"

	"github.com/GuoxinL/snowflake-gorm/nodeid"
	"github.com/GuoxinL/snowflake-gorm/nodeid/gorm/model"
	"github.com/GuoxinL/snowflake-gorm/nodeid/gorm/model/dao"
	"github.com/bwmarrin/snowflake"
	"go.uber.org/atomic"
	"gorm.io/gorm"
)

var _ snowflake.TimeSynchronizer = new(TimeSynchronizer)
var _ snowflake.NodeIdAllocator = new(NodeIdAllocator)

// NodeIdAllocator gorm节点ID分配器
type NodeIdAllocator struct {
	ctx context.Context
	dao *dao.Query
	// nodeIdKey 节点id key
	nodeIdKey string

	// 时钟回拨容忍时间
	acceptableClockDrift time.Duration
	// 节点id抢占时间间隔
	nodeIdContentionInterval time.Duration
	// 节点id分配器
	snowflake.NodeIdAllocator

	logger Logger
}

// NewNodeIdAllocator 创建一个新的节点ID分配器
func NewNodeIdAllocator(ctx context.Context, db *gorm.DB, name string, port int,
	acceptableClockDrift, nodeIdContentionInterval time.Duration, logger Logger) *NodeIdAllocator {
	// 1. 查询当前节点ID
	nodeIdKey := GetNodeIdKey(name, port)

	return &NodeIdAllocator{
		ctx:                      ctx,
		dao:                      dao.Use(db),
		logger:                   logger,
		nodeIdKey:                nodeIdKey,
		acceptableClockDrift:     acceptableClockDrift,
		nodeIdContentionInterval: nodeIdContentionInterval,
		NodeIdAllocator:          nodeid.NewHashNodeIdAllocator(nodeIdKey),
	}
}

// Alloc 分配一个新的节点ID
func (m NodeIdAllocator) Alloc() (int64, error) {
	now := time.Now()
	nowMilli := now.UnixMilli()

	nodeId, err := m.NodeIdAllocator.Alloc()
	if err != nil {
		return 0, err
	}

	tab := m.dao.SnowflakeKv
	for {
		// 1. 查询当前节点ID是否存在
		var saved *model.SnowflakeKv
		saved, err = tab.WithContext(m.ctx).Where(tab.Key.Eq(m.nodeIdKey), tab.NodeID.Eq(nodeId)).First()
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 2. 如果不存在，则创建一个新的节点ID
				saved = &model.SnowflakeKv{
					Key:     m.nodeIdKey,
					NodeID:  nodeId,
					Time:    nowMilli,
					Created: &now,
					Updated: now,
				}

				if err = tab.WithContext(m.ctx).Create(saved); err != nil {
					return 0, err
				}
				return saved.NodeID, nil
			}
			return 0, err
		}

		// 2. 判断保存的时间是否大于当前时间
		if saved.Time > nowMilli {
			// 2.1 如果回拨小于N秒则等待
			if nowMilli-m.acceptableClockDrift.Microseconds() <= saved.Time {
				time.Sleep(m.acceptableClockDrift)
				return saved.NodeID, nil
			}

			// 2.2 如果保存的时间大于当前时间，则返回时钟回拨报错
			m.logger.Errorf("time is rollback, please check the local clock!!! current: %s, saved: %s",
				now.Format(time.RFC3339), time.UnixMilli(saved.Time).Format(time.RFC3339))
			// 2.3 节点id漂移
			nodeId, err = m.NodeIdAllocator.Migration(nodeId)
			if err != nil {
				return 0, err
			}
			continue
		}

		// 3. 如果当前时间 - 节点id抢占时间间隔还是大于保存的时间 则抢占节点id
		if nowMilli-m.nodeIdContentionInterval.Milliseconds() > saved.Time {
			saved.NodeID = nodeId
		}

		// 4. 如果保存的时间小于当前时间，则更新保存时间
		saved.Time = nowMilli
		saved.Created = nil
		saved.Updated = now
		if _, err = tab.WithContext(m.ctx).Where(tab.Key.Eq(m.nodeIdKey), tab.NodeID.Eq(nodeId)).
			Updates(saved); err != nil {
			return 0, err
		}
		return saved.NodeID, nil
	}
}

// TimeSynchronizer 时间同步器
type TimeSynchronizer struct {
	ctx       context.Context
	dao       *dao.Query
	ticker    *time.Ticker
	nodeIdKey string
	logger    Logger

	// 填充前缀，避免与前面字段发生伪共享
	_pad0 [56]byte

	// curr 独占整个缓存行
	curr atomic.Int64

	// 填充后缀，防止后续字段干扰
	_pad1 [56]byte
}

func NewTimeSynchronizer(ctx context.Context, db *gorm.DB, name string, port int, interval time.Duration, logger Logger) *TimeSynchronizer {
	nodeIdKey := GetNodeIdKey(name, port)

	return &TimeSynchronizer{
		ctx:       ctx,
		dao:       dao.Use(db),
		nodeIdKey: nodeIdKey,
		ticker:    time.NewTicker(interval),
		logger:    logger,
	}
}
func (m *TimeSynchronizer) Async(t int64) {
	last := m.curr.Load()
	if t > last+10 { // 10ms 阈值
		m.curr.Store(t)
	}
	//m.curr.Store(t)
}

func (m *TimeSynchronizer) Run() {
	go func(m *TimeSynchronizer) {
		for {
			select {
			case <-m.ticker.C:
				m.updateDB()
			case <-m.ctx.Done():
				m.logger.Info("time synchronizer is done")
				return
			}
		}
	}(m)
}

// updateDB 将当前时间同步到数据库
func (m *TimeSynchronizer) updateDB() {
	currentTime := m.curr.Load()
	if currentTime == 0 {
		return
	}

	snowflakeKv := model.SnowflakeKv{
		Key:     m.nodeIdKey,
		Time:    currentTime,
		Updated: time.Now(),
	}
	tab := m.dao.SnowflakeKv
	// 保存
	if _, err := tab.WithContext(m.ctx).Where().Updates(snowflakeKv); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			m.logger.Errorf("update time failed. error: %v", err)
		}
	}
}
