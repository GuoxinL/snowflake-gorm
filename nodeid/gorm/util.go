//
// Copyright (C) BABEC. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0
//

// Package route example
package gorm

import (
	"fmt"
	"net"
	"os"
)

type DeployType string

const (
	K8s      DeployType = "k8s"
	Docker   DeployType = "docker"
	Physical DeployType = "physical"
)

func (d DeployType) Is(typ DeployType) bool {
	return d == typ
}

func GetNodeIdKey(name string, port int) string {
	return fmt.Sprintf("%s_%s_%d_%s", name, GetIP(), port, GetDeployType())
}

// GetDeployType 获取部署类型
func GetDeployType() DeployType {
	// 检查是否在Kubernetes环境中
	if _, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
		return K8s
	}

	// 检查是否在Docker环境中
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return Docker
	}

	// 默认返回物理机环境
	return Physical
}

// GetIP 获取有效的网卡IP地址
func GetIP() string {
	// 优先从环境变量获取
	if podIP := os.Getenv("POD_IP"); podIP != "" {
		if net.ParseIP(podIP) != nil {
			return podIP
		}
	}

	// 获取所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		fmt.Println("Error: Unable to get network interfaces:", err)
		return ""
	}

	// 遍历所有网络接口，获取IP地址
	for _, iface := range interfaces {
		// 忽略未启用和回环接口
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		// 遍历接口的所有地址，查找IPv4地址
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// 检查是否是IPv4地址且非本地地址
			if ip != nil && ip.To4() != nil && !ip.IsPrivate() && !ip.IsLoopback() {
				return ip.String()
			}
		}
	}

	// 如果未找到公网IP，尝试返回内网IP
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// 返回第一个有效的IPv4地址
			if ip != nil && ip.To4() != nil && !ip.IsLoopback() {
				return ip.String()
			}
		}
	}

	return ""
}
