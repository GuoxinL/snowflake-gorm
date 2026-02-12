//
// Copyright (C) BABEC. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0
//

// Package nodeid 哈希节点ID分配器测试
package nodeid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewHashNodeIdAllocator 测试哈希节点ID分配器创建
func TestNewHashNodeIdAllocator(t *testing.T) {
	nodeIdKey := "test-key"
	allocator := NewHashNodeIdAllocator(nodeIdKey)
	assert.NotNil(t, allocator)
}

// TestHashNodeIdAllocator_Alloc_SameKey 测试相同key产生相同的节点ID
func TestHashNodeIdAllocator_Alloc_SameKey(t *testing.T) {
	allocator := NewHashNodeIdAllocator("test-key-123")

	// 多次分配，应该返回相同的节点ID
	firstNodeId, err := allocator.Alloc()
	assert.NoError(t, err)

	secondNodeId, err := allocator.Alloc()
	assert.NoError(t, err)

	assert.Equal(t, firstNodeId, secondNodeId)
}

// TestHashNodeIdAllocator_Alloc_DifferentKey 测试不同key产生不同的节点ID
func TestHashNodeIdAllocator_Alloc_DifferentKey(t *testing.T) {
	allocator1 := NewHashNodeIdAllocator("test-key-1")
	allocator2 := NewHashNodeIdAllocator("test-key-2")

	nodeId1, err := allocator1.Alloc()
	assert.NoError(t, err)

	nodeId2, err := allocator2.Alloc()
	assert.NoError(t, err)

	assert.NotEqual(t, nodeId1, nodeId2)
}

// TestHashNodeIdAllocator_Alloc_Range 测试节点ID在有效范围内
func TestHashNodeIdAllocator_Alloc_Range(t *testing.T) {
	keys := []string{
		"test-key-1",
		"test-key-2",
		"test-key-3",
		"pod-name-12345",
		"service-name-abc",
		"node-192-168-1-1",
	}

	for _, key := range keys {
		allocator := NewHashNodeIdAllocator(key)
		nodeId, err := allocator.Alloc()
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, nodeId, int64(0))
		assert.Less(t, nodeId, int64(1024))
	}
}

// TestHashNodeIdAllocator_Alloc_Consistency 测试哈希一致性
func TestHashNodeIdAllocator_Alloc_Consistency(t *testing.T) {
	key := "consistent-test-key"
	allocator := NewHashNodeIdAllocator(key)

	// 验证多次分配结果一致
	expectedNodeId, err := allocator.Alloc()
	assert.NoError(t, err)

	for i := 0; i < 10; i++ {
		nodeId, err := allocator.Alloc()
		assert.NoError(t, err)
		assert.Equal(t, expectedNodeId, nodeId)
	}
}

// TestHashNodeIdAllocator_Alloc_EmptyKey 测试空key
func TestHashNodeIdAllocator_Alloc_EmptyKey(t *testing.T) {
	allocator := NewHashNodeIdAllocator("")
	nodeId, err := allocator.Alloc()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, nodeId, int64(0))
	assert.Less(t, nodeId, int64(1024))
}

// TestHashNodeIdAllocator_Migration 测试节点ID漂移
func TestHashNodeIdAllocator_Migration(t *testing.T) {
	allocator := NewHashNodeIdAllocator("test-key")

	// 测试多个节点ID的漂移
	testNodeIds := []int64{0, 100, 512, 1023, 555}

	for _, oldNodeId := range testNodeIds {
		newNodeId, err := allocator.Migration(oldNodeId)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, newNodeId, int64(0))
		assert.Less(t, newNodeId, int64(1024))

		// 验证相同输入产生相同输出
		newNodeId2, err := allocator.Migration(oldNodeId)
		assert.NoError(t, err)
		assert.Equal(t, newNodeId, newNodeId2)
	}
}

// TestHashNodeIdAllocator_Migration_DifferentFromOriginal 测试漂移后的节点ID与原ID不同
func TestHashNodeIdAllocator_Migration_DifferentFromOriginal(t *testing.T) {
	allocator := NewHashNodeIdAllocator("test-key-migration")

	// 大多数情况下，漂移后的ID应该与原ID不同
	diffCount := 0
	for i := 0; i < 100; i++ {
		oldNodeId := int64(i % 1024)
		newNodeId, err := allocator.Migration(oldNodeId)
		assert.NoError(t, err)
		if newNodeId != oldNodeId {
			diffCount++
		}
	}
	// 至少有一些ID发生了漂移
	assert.Greater(t, diffCount, 50)
}

// TestHashNodeIdAllocator_Migration_Consistency 测试漂移结果一致性
func TestHashNodeIdAllocator_Migration_Consistency(t *testing.T) {
	allocator := NewHashNodeIdAllocator("consistency-test")

	oldNodeId := int64(123)

	// 多次漂移应该产生相同的结果
	newNodeId1, err := allocator.Migration(oldNodeId)
	assert.NoError(t, err)

	newNodeId2, err := allocator.Migration(oldNodeId)
	assert.NoError(t, err)

	newNodeId3, err := allocator.Migration(oldNodeId)
	assert.NoError(t, err)

	assert.Equal(t, newNodeId1, newNodeId2)
	assert.Equal(t, newNodeId2, newNodeId3)
}
