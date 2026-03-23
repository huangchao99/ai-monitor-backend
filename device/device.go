package device

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

const (
	cidPath      = "/sys/class/mmc_host/mmc0/mmc0:0001/cid"
	persistPath  = "/opt/device_id"
)

var (
	once     sync.Once
	deviceID string
)

// Get 返回设备唯一标识（SHA256 of eMMC CID）。
// 第一次调用时读取或生成，之后直接返回缓存值。
func Get() string {
	once.Do(func() {
		deviceID = loadOrGenerate()
	})
	return deviceID
}

// Init 在服务启动时调用，提前完成 ID 初始化并打印日志。
func Init() {
	id := Get()
	log.Printf("[device] DeviceID: %s", id)
}

func loadOrGenerate() string {
	// 优先从持久化文件读取
	if data, err := os.ReadFile(persistPath); err == nil {
		id := strings.TrimSpace(string(data))
		if id != "" {
			return id
		}
	}

	// 读取 eMMC CID 并生成 SHA256
	id, err := generateFromCID()
	if err != nil {
		log.Printf("[device] WARNING: 无法读取 eMMC CID (%v)，使用空 device_id", err)
		return ""
	}

	// 持久化写入 /opt/device_id
	if err := os.WriteFile(persistPath, []byte(id), 0644); err != nil {
		log.Printf("[device] WARNING: 写入 %s 失败: %v", persistPath, err)
	} else {
		log.Printf("[device] DeviceID 已生成并写入 %s", persistPath)
	}
	return id
}

func generateFromCID() (string, error) {
	data, err := os.ReadFile(cidPath)
	if err != nil {
		return "", fmt.Errorf("读取 %s 失败: %w", cidPath, err)
	}
	cid := strings.TrimSpace(string(data))
	if cid == "" {
		return "", fmt.Errorf("eMMC CID 为空")
	}
	sum := sha256.Sum256([]byte(cid))
	return fmt.Sprintf("%x", sum), nil
}
