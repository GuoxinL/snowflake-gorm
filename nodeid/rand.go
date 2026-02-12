//
// Copyright (C) BABEC. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0
//

// Package nodeid 随机节点ID分配器
package nodeid

import (
	"math/rand/v2"

	"github.com/bwmarrin/snowflake"
)

// RandNodeIdAllocator 随机节点ID分配器
type RandNodeIdAllocator struct {
}

// NewRandNodeIdAllocator 创建一个随机节点ID分配器
// @return snowflake.NodeIdAllocator
func NewRandNodeIdAllocator() snowflake.NodeIdAllocator {
	return &RandNodeIdAllocator{}
}

// Alloc 分配一个随机节点ID
// @receiver n
// @return nodeId
// @return err
func (n *RandNodeIdAllocator) Alloc() (nodeId int64, err error) {
	return rand.Int64N(1023), nil
}

// Migration 节点ID漂移
// @receiver n
// @param nodeId
// @return newNodeId
// @return err
func (n *RandNodeIdAllocator) Migration(_ int64) (newNodeId int64, err error) {
	return rand.Int64N(1023), nil
}
