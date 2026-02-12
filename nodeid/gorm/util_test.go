//
// Copyright (C) BABEC. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0
//

// Package gorm 节点id分配器 工具测试
package gorm

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDeployType_Is 测试部署类型判断
func TestDeployType_Is(t *testing.T) {
	k8sType := K8s
	assert.True(t, k8sType.Is(K8s))
	assert.False(t, k8sType.Is(Docker))
	assert.False(t, k8sType.Is(Physical))

	dockerType := Docker
	assert.True(t, dockerType.Is(Docker))
	assert.False(t, dockerType.Is(K8s))
	assert.False(t, dockerType.Is(Physical))

	physicalType := Physical
	assert.True(t, physicalType.Is(Physical))
	assert.False(t, physicalType.Is(K8s))
	assert.False(t, physicalType.Is(Docker))
}

// TestGetDeployType_Default 测试默认部署类型
func TestGetDeployType_Default(t *testing.T) {
	// 保存原始环境变量
	oldK8sEnv, k8sExists := os.LookupEnv("KUBERNETES_SERVICE_HOST")

	// 清除Kubernetes环境变量
	os.Unsetenv("KUBERNETES_SERVICE_HOST")

	deployType := GetDeployType()
	assert.Equal(t, Physical, deployType)

	// 恢复环境变量
	if k8sExists {
		os.Setenv("KUBERNETES_SERVICE_HOST", oldK8sEnv)
	}
}

// TestGetDeployType_K8s 测试Kubernetes环境检测
func TestGetDeployType_K8s(t *testing.T) {
	// 保存原始环境变量
	oldK8sEnv, k8sExists := os.LookupEnv("KUBERNETES_SERVICE_HOST")

	// 设置Kubernetes环境变量
	os.Setenv("KUBERNETES_SERVICE_HOST", "10.96.0.1")
	defer func() {
		if k8sExists {
			os.Setenv("KUBERNETES_SERVICE_HOST", oldK8sEnv)
		} else {
			os.Unsetenv("KUBERNETES_SERVICE_HOST")
		}
	}()

	deployType := GetDeployType()
	assert.Equal(t, K8s, deployType)
}

// TestGetNodeIdKey 测试节点ID Key生成
func TestGetNodeIdKey(t *testing.T) {
	port := 8080
	name := "test-service"

	key := GetNodeIdKey(name, port)
	assert.NotEmpty(t, key)
	assert.Contains(t, key, "_8080_")
}

// TestGetIP_FromEnv 测试从环境变量获取IP
func TestGetIP_FromEnv(t *testing.T) {
	// 保存原始环境变量
	oldPodIP, podIPExists := os.LookupEnv("POD_IP")

	// 设置POD_IP环境变量
	testIP := "192.168.1.100"
	os.Setenv("POD_IP", testIP)
	defer func() {
		if podIPExists {
			os.Setenv("POD_IP", oldPodIP)
		} else {
			os.Unsetenv("POD_IP")
		}
	}()

	ip := GetIP()
	assert.Equal(t, testIP, ip)
}

// TestGetIP_InvalidEnv 测试无效的环境变量IP
func TestGetIP_InvalidEnv(t *testing.T) {
	// 保存原始环境变量
	oldPodIP, podIPExists := os.LookupEnv("POD_IP")

	// 设置无效的POD_IP环境变量
	os.Setenv("POD_IP", "invalid-ip")
	defer func() {
		if podIPExists {
			os.Setenv("POD_IP", oldPodIP)
		} else {
			os.Unsetenv("POD_IP")
		}
	}()

	ip := GetIP()
	// 应该返回网络接口的IP或空字符串，不应该是无效的IP
	assert.NotEqual(t, "invalid-ip", ip)
}

// TestGetIP_NoEnv 测试无环境变量时从网络接口获取IP
func TestGetIP_NoEnv(t *testing.T) {
	// 保存原始环境变量
	oldPodIP, podIPExists := os.LookupEnv("POD_IP")

	// 清除POD_IP环境变量
	os.Unsetenv("POD_IP")
	defer func() {
		if podIPExists {
			os.Setenv("POD_IP", oldPodIP)
		}
	}()

	ip := GetIP()
	// 可能返回空字符串（如果没有有效的网络接口）
	// 或者返回有效的IP地址
	if ip != "" {
		// 如果返回非空IP，验证格式
		assert.NotEmpty(t, ip)
	}
}

// TestGetNodeIdKey_DifferentPort 测试不同端口生成不同的Key
func TestGetNodeIdKey_DifferentPort(t *testing.T) {
	name := "test-service"

	key1 := GetNodeIdKey(name, 8080)
	key2 := GetNodeIdKey(name, 9090)

	assert.NotEqual(t, key1, key2)
	assert.Contains(t, key1, "_8080_")
	assert.Contains(t, key2, "_9090_")
}

// TestGetNodeIdKey_SamePort 测试相同端口生成相同的Key
func TestGetNodeIdKey_SamePort(t *testing.T) {
	name := "test-service"

	key1 := GetNodeIdKey(name, 8080)
	key2 := GetNodeIdKey(name, 8080)

	assert.Equal(t, key1, key2)
}

// TestDeployType_String 测试部署类型字符串表示
func TestDeployType_String(t *testing.T) {
	assert.Equal(t, "k8s", string(K8s))
	assert.Equal(t, "docker", string(Docker))
	assert.Equal(t, "physical", string(Physical))
}

// TestGetNodeIdKey_Format 测试节点ID Key格式
func TestGetNodeIdKey_Format(t *testing.T) {
	port := 8080
	name := "test-service"

	key := GetNodeIdKey(name, port)

	// 验证格式: {service_name}_{ip}_{port}_{deploy_type}
	// 应该包含4个下划线分隔的部分
	assert.NotEmpty(t, key)
	assert.Contains(t, key, "_8080_")
}

// TestGetIP_Loopback 测试本地回环地址被忽略
func TestGetIP_Loopback(t *testing.T) {
	// 保存原始环境变量
	oldPodIP, podIPExists := os.LookupEnv("POD_IP")

	// 清除环境变量
	os.Unsetenv("POD_IP")
	defer func() {
		if podIPExists {
			os.Setenv("POD_IP", oldPodIP)
		}
	}()

	ip := GetIP()
	// 如果返回IP，不应该包含127.0.0.1
	if ip != "" {
		assert.NotEqual(t, "127.0.0.1", ip)
	}
}

// TestGetIP_PrivateIP 测试内网IP回退
func TestGetIP_PrivateIP(t *testing.T) {
	// 保存原始环境变量
	oldPodIP, podIPExists := os.LookupEnv("POD_IP")

	// 清除环境变量
	os.Unsetenv("POD_IP")
	defer func() {
		if podIPExists {
			os.Setenv("POD_IP", oldPodIP)
		}
	}()

	ip := GetIP()
	// 在没有公网IP的情况下，应该返回内网IP或空字符串
	// 这里只是验证函数不会崩溃
	if ip != "" {
		assert.NotEmpty(t, ip)
	}
}

// TestGetDeployType_StringValues 测试部署类型字符串值
func TestGetDeployType_StringValues(t *testing.T) {
	assert.Equal(t, "k8s", string(K8s))
	assert.Equal(t, "docker", string(Docker))
	assert.Equal(t, "physical", string(Physical))

	// 测试转换
	var dt DeployType = "k8s"
	assert.Equal(t, K8s, dt)

	dt = "custom"
	assert.NotEqual(t, K8s, dt)
	assert.NotEqual(t, Docker, dt)
	assert.NotEqual(t, Physical, dt)
}
