//
// Copyright (C) BABEC. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0
//

// Package nodeid 哈希节点ID分配器
package nodeid

import (
	"encoding/binary"

	"github.com/bwmarrin/snowflake"
	xxhash2 "github.com/cespare/xxhash/v2"
)

// HashNodeIdAllocator 哈希节点ID分配器
type HashNodeIdAllocator struct {
	nodeIdKey string
}

// NewHashNodeIdAllocator 创建一个哈希节点ID分配器
// @param nodeIdKey
// @return snowflake.NodeIdAllocator
func NewHashNodeIdAllocator(nodeIdKey string) snowflake.NodeIdAllocator {
	return &HashNodeIdAllocator{nodeIdKey: nodeIdKey}
}

// Alloc 分配一个哈希节点ID
// @receiver n
// @return nodeId
// @return err
func (n *HashNodeIdAllocator) Alloc() (int64, error) {
	d := xxhash2.New()
	_, _ = d.WriteString(n.nodeIdKey)
	return int64(d.Sum64() % 1024), nil
}

func (n *HashNodeIdAllocator) Migration(nodeId int64) (newNodeId int64, err error) {
	nodeIdBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(nodeIdBytes, uint64(nodeId))
	d := xxhash2.New()
	_, _ = d.Write(nodeIdBytes)
	return int64(d.Sum64() % 1024), nil
}
