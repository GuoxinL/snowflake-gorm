//
// Copyright (C) BABEC. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0
//

// Package nodeid 测试随机节点ID分配器
package nodeid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewRandNodeIdAllocator 测试随机节点ID分配器创建
func TestNewRandNodeIdAllocator(t *testing.T) {
	allocator := NewRandNodeIdAllocator()
	assert.NotNil(t, allocator)
}

// TestRandNodeIdAllocator_Alloc_Range 测试随机节点ID在有效范围内
func TestRandNodeIdAllocator_Alloc_Range(t *testing.T) {
	allocator := NewRandNodeIdAllocator()

	for i := 0; i < 100; i++ {
		nodeId, err := allocator.Alloc()
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, nodeId, int64(0))
		assert.LessOrEqual(t, nodeId, int64(1023))
	}
}

// TestRandNodeIdAllocator_Alloc_Different 测试随机分配产生不同的节点ID
func TestRandNodeIdAllocator_Alloc_Different(t *testing.T) {
	allocator := NewRandNodeIdAllocator()

	// 收集100个节点ID
	nodeIds := make(map[int64]bool)
	for i := 0; i < 100; i++ {
		nodeId, err := allocator.Alloc()
		assert.NoError(t, err)
		nodeIds[nodeId] = true
	}

	// 验证至少有一些不同的节点ID（极大概率）
	assert.Greater(t, len(nodeIds), 10)
}

// TestRandNodeIdAllocator_Alloc_DifferentAllocators 测试不同分配器产生不同的节点ID
func TestRandNodeIdAllocator_Alloc_DifferentAllocators(t *testing.T) {
	allocator1 := NewRandNodeIdAllocator()
	allocator2 := NewRandNodeIdAllocator()

	// 多次分配
	nodeIds1 := make([]int64, 10)
	nodeIds2 := make([]int64, 10)

	for i := 0; i < 10; i++ {
		nodeId, err := allocator1.Alloc()
		assert.NoError(t, err)
		nodeIds1[i] = nodeId

		nodeId, err = allocator2.Alloc()
		assert.NoError(t, err)
		nodeIds2[i] = nodeId
	}

	// 验证两个分配器的结果不同（极大概率）
	allEqual := true
	for i := 0; i < 10; i++ {
		if nodeIds1[i] != nodeIds2[i] {
			allEqual = false
			break
		}
	}
	assert.False(t, allEqual)
}

// TestRandNodeIdAllocator_Migration_Range 测试节点ID漂移范围
func TestRandNodeIdAllocator_Migration_Range(t *testing.T) {
	allocator := NewRandNodeIdAllocator()

	testNodeIds := []int64{0, 100, 512, 1023, 555}

	for _, oldNodeId := range testNodeIds {
		newNodeId, err := allocator.Migration(oldNodeId)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, newNodeId, int64(0))
		assert.LessOrEqual(t, newNodeId, int64(1023))
	}
}

// TestRandNodeIdAllocator_Migration_Different 测试漂移产生不同的节点ID
func TestRandNodeIdAllocator_Migration_Different(t *testing.T) {
	allocator := NewRandNodeIdAllocator()

	oldNodeId := int64(100)

	// 多次漂移
	newNodeIds := make(map[int64]bool)
	for i := 0; i < 50; i++ {
		newNodeId, err := allocator.Migration(oldNodeId)
		assert.NoError(t, err)
		newNodeIds[newNodeId] = true
	}

	// 随机漂移应该产生多个不同的结果（极大概率）
	assert.Greater(t, len(newNodeIds), 30)
}

// TestRandNodeIdAllocator_Migration_DifferentFromOriginal 测试漂移后的节点ID可能与原ID不同
func TestRandNodeIdAllocator_Migration_DifferentFromOriginal(t *testing.T) {
	allocator := NewRandNodeIdAllocator()

	oldNodeId := int64(500)

	// 多次漂移，至少有一些结果与原ID不同
	diffCount := 0
	for i := 0; i < 100; i++ {
		newNodeId, err := allocator.Migration(oldNodeId)
		assert.NoError(t, err)
		if newNodeId != oldNodeId {
			diffCount++
		}
	}
	// 极大概率会有不同的结果
	assert.Greater(t, diffCount, 50)
}
