package systeminfo

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// NetworkDeviceInfo 网络设备信息结构体，包含所有需要查询的字段
type NetworkDeviceInfo struct {
	DeviceName  string // 设备名（如 eth0、ens33）
	ConnName    string // 对应的连接名（nmcli 核心配置单元）
	AddressMode string // 地址模式：static（静态）/ dhcp（自动获取）
	IPAddress   string // IPv4 地址
	SubnetMask  string // 子网掩码（如 255.255.255.0）
	Gateway     string // 默认网关
	DNS         string // 主 DNS 服务器
	BackupDNS   string // 备用 DNS 服务器
	Status      int    // 100 已连接 10未托管 30未连接 20 不可用
}

// execCommand 执行系统命令，返回标准输出和错误（合并 stderr）
func execCommand(cmd string, args ...string) (string, error) {
	command := exec.Command(cmd, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("execCommand failed: %v, output: %s", err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

// -------------------------- 函数1：查询所有网络设备信息 --------------------------
// GetAllNetworkDevices 获取系统中所有网络设备的完整信息（静态/DHCP、IP、掩码、网关、DNS）
// func GetAllNetworkDevices() ([]NetworkDeviceInfo, error) {
// 	var devices []NetworkDeviceInfo

// 	// 步骤1：获取所有设备名 + 对应的连接名（nmcli 简洁输出，冒号分隔）
// 	deviceConnOutput, err := execCommand("nmcli", "-t", "-f", "DEVICE,CONNECTION", "device", "status")
// 	if err != nil {
// 		return nil, fmt.Errorf("get dev failed: %v", err)
// 	}
// 	if deviceConnOutput == "" {
// 		return nil, errors.New("can not find any network device")
// 	}

// 	// 遍历每个设备，解析设备名和连接名
// 	deviceLines := strings.Split(deviceConnOutput, "\n")
// 	for _, line := range deviceLines {
// 		if line == "" {
// 			continue
// 		}
// 		// 分割设备名和连接名（格式：设备名:连接名）
// 		parts := strings.Split(line, ":")
// 		if len(parts) != 2 {
// 			continue
// 		}
// 		deviceName := strings.TrimSpace(parts[0])
// 		connName := strings.TrimSpace(parts[1])
// 		// 过滤无连接的设备（如 lo 回环、未连接的网卡）
// 		if connName == "" || connName == "--" {
// 			continue
// 		}

// 		// 步骤2：获取该连接的详细网络配置（ipv4 核心字段）
// 		connInfoOutput, err := execCommand("nmcli", "-t", "-f", "ipv4.method,ipv4.addresses,ipv4.gateway,ipv4.dns", "connection", "show", connName)
// 		if err != nil {
// 			return nil, fmt.Errorf("get conn[%s]  : %v", connName, err)
// 		}

// 		// 初始化设备信息
// 		deviceInfo := NetworkDeviceInfo{
// 			DeviceName: deviceName,
// 			ConnName:   connName,
// 		}

// 		// 解析连接配置的每一行（格式：字段名:值）
// 		connLines := strings.Split(connInfoOutput, "\n")
// 		for _, connLine := range connLines {
// 			if connLine == "" {
// 				continue
// 			}
// 			kv := strings.SplitN(connLine, ":", 2)
// 			if len(kv) != 2 {
// 				continue
// 			}
// 			key := strings.TrimSpace(kv[0])
// 			value := strings.TrimSpace(kv[1])

// 			// 匹配字段并赋值
// 			switch key {
// 			case "ipv4.method":
// 				// 转换 nmcli 模式为业务模式：auto=DHCP，manual=static
// 				if value == "auto" {
// 					deviceInfo.AddressMode = "dhcp"
// 				} else if value == "manual" {
// 					deviceInfo.AddressMode = "static"
// 				} else {
// 					deviceInfo.AddressMode = "unknown"
// 				}
// 			case "ipv4.addresses":
// 				// 解析 CIDR 为 IP + 子网掩码
// 				ip, mask, err := parseCIDR(value)
// 				if err == nil {
// 					deviceInfo.IPAddress = ip
// 					deviceInfo.SubnetMask = mask
// 				}
// 			case "ipv4.gateway":
// 				deviceInfo.Gateway = value
// 			case "ipv4.dns":
// 				// 拆分主 DNS 和备用 DNS（空格分隔）
// 				dnsParts := strings.Fields(value)
// 				if len(dnsParts) >= 1 {
// 					deviceInfo.DNS = dnsParts[0]
// 				}
// 				if len(dnsParts) >= 2 {
// 					deviceInfo.BackupDNS = dnsParts[1]
// 				}
// 			}
// 		}

// 		devices = append(devices, deviceInfo)
// 	}

// 	return devices, nil
// }

// -------------------------- 函数2：配置指定设备的网络参数 --------------------------
// ConfigNetworkDevice 配置指定设备的网络（地址模式、IP、掩码、网关、DNS、备用DNS）
// 参数说明：
//
//	deviceName: 设备名（如 eth0）
//	mode: 地址模式，仅支持 "static" 或 "dhcp"
//	ip: 静态模式下的 IP 地址（dhcp 模式可传空）
//	mask: 静态模式下的子网掩码（dhcp 模式可传空）
//	gateway: 静态模式下的默认网关（dhcp 模式可传空）
//	dns: 主 DNS 服务器（可传空）
//	backupDns: 备用 DNS 服务器（可传空）
// func ConfigNetworkDevice(deviceName, mode, ip, mask, gateway, dns, backupDns string) error {
// 	// 步骤1：参数合法性校验
// 	if deviceName == "" {
// 		return errors.New("device name can not be empty")
// 	}
// 	mode = strings.ToLower(mode)
// 	if mode != "static" && mode != "dhcp" {
// 		return fmt.Errorf("device mode only support static or dhcp, mode %s invalid", mode) //errors.New("device mode only support static or dhcp, mode %s invalid")
// 	}
// 	// 静态模式下，核心参数不能为空
// 	if mode == "static" {
// 		if ip == "" || mask == "" || gateway == "" {
// 			return errors.New("static config, ip/mask/gateway can not be empty")
// 		}
// 	}

// 	// 步骤2：根据设备名查找对应的连接名
// 	deviceConnOutput, err := execCommand("nmcli", "-t", "-f", "DEVICE,CONNECTION", "device", "status")
// 	if err != nil {
// 		return fmt.Errorf("get device status failed: %v", err)
// 	}
// 	var connName string
// 	deviceLines := strings.Split(deviceConnOutput, "\n")
// 	for _, line := range deviceLines {
// 		if line == "" {
// 			continue
// 		}
// 		parts := strings.Split(line, ":")
// 		if len(parts) != 2 {
// 			continue
// 		}
// 		if strings.TrimSpace(parts[0]) == deviceName {
// 			connName = strings.TrimSpace(parts[1])
// 			break
// 		}
// 	}
// 	if connName == "" || connName == "--" {
// 		return fmt.Errorf("device [%s] can not find", deviceName)
// 	}

// 	// 步骤3：拼接 nmcli 修改命令
// 	var modifyArgs []string
// 	modifyArgs = append(modifyArgs, "nmcli", "connection", "modify", connName)

// 	// 配置地址模式 + 对应参数
// 	if mode == "static" {
// 		// 静态模式：转换 IP+掩码 为 CIDR 格式
// 		cidr := fmt.Sprintf("%s/%s", ip, subnetMaskToPrefix(mask))
// 		modifyArgs = append(modifyArgs,
// 			"ipv4.method", "manual",
// 			"ipv4.addresses", cidr,
// 			"ipv4.gateway", gateway,
// 		)
// 	} else {
// 		// DHCP 模式：清空静态配置，设置自动获取
// 		modifyArgs = append(modifyArgs,
// 			"ipv4.method", "auto",
// 			"ipv4.addresses", "",
// 			"ipv4.gateway", "",
// 		)
// 	}

// 	// 配置 DNS（主+备用，空格分隔）
// 	dnsStr := ""
// 	if dns != "" {
// 		dnsStr = dns
// 		if backupDns != "" {
// 			dnsStr += " " + backupDns
// 		}
// 	}
// 	modifyArgs = append(modifyArgs, "ipv4.dns", dnsStr)

// 	// 步骤4：执行修改命令（需 root 权限）
// 	_, err = execCommand("sudo", modifyArgs...)
// 	if err != nil {
// 		return fmt.Errorf("modify network failed: %v", err)
// 	}

// 	// 步骤5：重新激活连接，使配置生效
// 	_, err = execCommand("sudo", "nmcli", "connection", "up", connName)
// 	if err != nil {
// 		return fmt.Errorf("nmcli connection up failed: %v", err)
// 	}

// 	fmt.Printf("device[%s] network config success, current mode: %s\n", deviceName, mode)
// 	return nil
// }

// -------------------------- 工具函数：子网掩码转 CIDR 前缀 --------------------------
// subnetMaskToPrefix 将子网掩码（如 255.255.255.0）转换为 CIDR 前缀（如 24）
func subnetMaskToPrefix(mask string) string {
	parts := strings.Split(mask, ".")
	if len(parts) != 4 {
		return "24" // 默认 /24
	}
	prefix := 0
	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil || num < 0 || num > 255 {
			return "24"
		}
		// 计算每个字节的二进制 1 的个数
		prefix += strings.Count(fmt.Sprintf("%08b", num), "1")
	}
	// 确保前缀在合法范围（0-32）
	if prefix < 0 || prefix > 32 {
		return "24"
	}
	return strconv.Itoa(prefix)
}

// -------------------------- 测试示例 --------------------------
func main() {
	// 示例1：查询所有网络设备信息
	fmt.Println("=== 查询所有网络设备信息 ===")
	devices, err := GetAllNetworkDevices()
	if err != nil {
		fmt.Printf("查询失败: %v\n", err)
		return
	}
	for _, dev := range devices {
		fmt.Printf("设备名: %s\n", dev.DeviceName)
		fmt.Printf("连接名: %s\n", dev.ConnName)
		fmt.Printf("地址模式: %s\n", dev.AddressMode)
		fmt.Printf("IP地址: %s\n", dev.IPAddress)
		fmt.Printf("子网掩码: %s\n", dev.SubnetMask)
		fmt.Printf("默认网关: %s\n", dev.Gateway)
		fmt.Printf("主DNS: %s\n", dev.DNS)
		fmt.Printf("备用DNS: %s\n", dev.BackupDNS)
		fmt.Println("------------------------")
	}

	// 示例2：配置指定设备为静态IP（需替换为你的设备名和参数）
	// fmt.Println("\n=== 配置设备网络为静态IP ===")
	// err = ConfigNetworkDevice(
	// 	"eth0",       // 设备名
	// 	"static",     // 地址模式
	// 	"192.168.1.100", // IP地址
	// 	"255.255.255.0", // 子网掩码
	// 	"192.168.1.1",   // 网关
	// 	"8.8.8.8",       // 主DNS
	// 	"114.114.114.114", // 备用DNS
	// )
	// if err != nil {
	// 	fmt.Printf("配置失败: %v\n", err)
	// 	return
	// }

	// 示例3：配置指定设备为DHCP（需替换为你的设备名）
	// fmt.Println("\n=== 配置设备网络为DHCP ===")
	// err = ConfigNetworkDevice(
	// 	"eth0",   // 设备名
	// 	"dhcp",   // 地址模式
	// 	"", "", "", "", "", // 静态参数传空
	// )
	// if err != nil {
	// 	fmt.Printf("配置失败: %v\n", err)
	// 	return
	// }
}

/////////////////////////////////////////////////

// -------------------------- 工具函数：CIDR 转 IP + 子网掩码 --------------------------
// parseCIDR 将 CIDR 格式（如 192.168.1.100/24）解析为 IP 地址和子网掩码
func parseCIDR2(cidr string) (ip string, mask string, err error) {
	if cidr == "" || cidr == "--" {
		return "", "", nil
	}
	fmt.Println("parseCIDR:", cidr)
	// 解析 CIDR
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", "", fmt.Errorf("parse CIDR failed: %v", err)
	}
	// 提取 IP 地址
	ip = ipNet.IP.String()
	// 转换子网掩码（字节转十进制字符串）
	maskBytes := ipNet.Mask
	mask = fmt.Sprintf("%d.%d.%d.%d", maskBytes[0], maskBytes[1], maskBytes[2], maskBytes[3])
	return ip, mask, nil
}

func parseCIDR(cidr string) (ip string, mask string, err error) {
	if cidr == "" || cidr == "--" {
		return "", "", nil
	}
	fmt.Println("parseCIDR:", cidr)

	// 步骤1：拆分 CIDR 为 原始IP 和 前缀（如 192.168.2.81/23 → 192.168.2.81 和 23）
	parts := strings.Split(cidr, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid CIDR format: %s (need ip/prefix)", cidr)
	}
	originalIP := parts[0] // 提取原始IP：192.168.2.81

	// 步骤2：验证原始IP是否合法
	if net.ParseIP(originalIP) == nil {
		return "", "", fmt.Errorf("invalid IP address: %s", originalIP)
	}

	// 步骤3：解析 CIDR 获取子网掩码（仍用原方法，只取掩码，忽略网段IP）
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", "", fmt.Errorf("parse CIDR failed: %v", err)
	}

	// 步骤4：原始IP作为返回的IP，从ipNet中提取掩码
	ip = originalIP
	maskBytes := ipNet.Mask
	mask = fmt.Sprintf("%d.%d.%d.%d", maskBytes[0], maskBytes[1], maskBytes[2], maskBytes[3])

	return ip, mask, nil
}

// -------------------------- 工具函数：解析 IP4.ADDRESS 字段 --------------------------
// parseIP4Address 解析 IP4.ADDRESS[1]: 192.168.2.81/23 格式的字符串，返回 IP 和子网掩码
func parseIP4Address(addressLine string) (ip string, mask string, err error) {
	// 正则匹配 CIDR 部分（如 192.168.2.81/23）
	regex := regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+/\d+)`)
	matches := regex.FindStringSubmatch(addressLine)
	if len(matches) < 1 {
		return "", "", errors.New("invalid IP4.ADDRESS format")
	}
	return parseCIDR(matches[1])
}

// -------------------------- 工具函数：解析 IP4.DNS 字段 --------------------------
// parseIP4DNS 解析 IP4.DNS[1]:192.168.2.1 这类多行 DNS 字段，返回主DNS和备用DNS
func parseIP4DNS(dnsLines []string) (primaryDNS, backupDNS string) {
	if len(dnsLines) >= 1 {
		primaryDNS = strings.TrimSpace(strings.Split(dnsLines[0], ":")[1])
	}
	if len(dnsLines) >= 2 {
		backupDNS = strings.TrimSpace(strings.Split(dnsLines[1], ":")[1])
	}
	return
}

func parseGeneralState(stateStr string) (int, error) {
	var stateNumRegex = regexp.MustCompile(`GENERAL\.STATE:\s*(\d+)`)
	// 用正则匹配数字部分
	matches := stateNumRegex.FindStringSubmatch(stateStr)
	if len(matches) < 2 {
		return 0, fmt.Errorf("invalid GENERAL.STATE format: %s", stateStr)
	}

	// 将匹配到的数字字符串转为int
	numStr := matches[1]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("convert state num to int failed: %v, numStr: %s", err, numStr)
	}

	return num, nil
}

// -------------------------- 函数1：查询所有网络设备信息 --------------------------
// GetAllNetworkDevices 获取系统中所有网络设备的完整信息（静态/DHCP、IP、掩码、网关、DNS）
// 优化点：
// 1. 优先从 nmcli device show 获取动态状态（IP4.ADDRESS/IP4.GATEWAY/IP4.DNS）
// 2. 从 nmcli connection show 获取地址模式（static/dhcp）
// 3. 兼容空值和异常格式
func GetAllNetworkDevices() ([]NetworkDeviceInfo, error) {
	var devices []NetworkDeviceInfo

	// 步骤1：获取所有设备名 + 对应的连接名（nmcli 简洁输出，冒号分隔）
	deviceConnOutput, err := execCommand("nmcli", "-t", "-f", "DEVICE,CONNECTION", "device", "status")
	if err != nil {
		return nil, fmt.Errorf("get dev failed: %v", err)
	}
	if deviceConnOutput == "" {
		return nil, errors.New("can not find any network device")
	}

	// 遍历每个设备，解析设备名和连接名
	deviceLines := strings.Split(deviceConnOutput, "\n")
	for _, line := range deviceLines {
		if line == "" {
			continue
		}
		// 分割设备名和连接名（格式：设备名:连接名）
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		deviceName := strings.TrimSpace(parts[0])
		connName := strings.TrimSpace(parts[1])
		// 过滤无连接的设备（如 lo 回环、未连接的网卡）
		if connName == "" || connName == "--" {
			continue
		}

		// 初始化设备信息
		deviceInfo := NetworkDeviceInfo{
			DeviceName: deviceName,
			ConnName:   connName,
		}

		fmt.Println(">>>>>start to get:", deviceName)

		// 步骤2：获取该设备的动态状态（IP4.ADDRESS/IP4.GATEWAY/IP4.DNS）
		deviceShowOutput, err := execCommand("nmcli", "device", "show", deviceName)
		if err == nil && deviceShowOutput != "" {
			deviceShowLines := strings.Split(deviceShowOutput, "\n")
			var dnsLines []string
			for _, devLine := range deviceShowLines {
				devLine = strings.TrimSpace(devLine)
				if devLine == "" {
					continue
				}

				// parse status
				if strings.HasPrefix(devLine, "GENERAL.STATE") {
					deviceInfo.Status, err = parseGeneralState(devLine)

				}
				// 解析 IP4.ADDRESS
				if strings.HasPrefix(devLine, "IP4.ADDRESS[") {
					ip, mask, _ := parseIP4Address(devLine)
					if ip != "" {
						deviceInfo.IPAddress = ip
						deviceInfo.SubnetMask = mask
						fmt.Println(">get ip:", ip, ",mask:", mask, devLine)
					}
				}
				// 解析 IP4.GATEWAY
				if strings.HasPrefix(devLine, "IP4.GATEWAY:") {
					deviceInfo.Gateway = strings.TrimSpace(strings.Split(devLine, ":")[1])
				}
				// 收集所有 IP4.DNS 行
				if strings.HasPrefix(devLine, "IP4.DNS[") {
					dnsLines = append(dnsLines, devLine)
				}
			}
			// 解析 DNS
			deviceInfo.DNS, deviceInfo.BackupDNS = parseIP4DNS(dnsLines)
		}

		// 步骤3：获取该连接的地址模式（static/dhcp）
		connInfoOutput, err := execCommand("nmcli", "-t", "-f", "ipv4.method", "connection", "show", connName)
		if err == nil && connInfoOutput != "" {
			connLine := strings.TrimSpace(connInfoOutput)
			kv := strings.SplitN(connLine, ":", 2)
			if len(kv) != 2 {
				continue
			}
			//key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			if value == "auto" {
				deviceInfo.AddressMode = "dhcp"
			} else if value == "manual" {
				deviceInfo.AddressMode = "static"
			} else {
				deviceInfo.AddressMode = "unknown"
			}
		}

		devices = append(devices, deviceInfo)
	}

	return devices, nil
}

// -------------------------- 函数2：配置指定设备的网络参数 --------------------------
// ConfigNetworkDevice 配置指定设备的网络（地址模式、IP、掩码、网关、DNS、备用DNS）
// 优化点：
// 1. 保留原配置逻辑（nmcli connection modify），兼容新的字段格式
// 2. 增强参数校验（子网掩码转前缀的容错）
// 3. 优化错误提示
func ConfigNetworkDevice(deviceName, mode, ip, mask, gateway, dns, backupDns string) error {
	// 步骤1：参数合法性校验
	if deviceName == "" {
		return errors.New("device name can not be empty")
	}
	mode = strings.ToLower(mode)
	if mode != "static" && mode != "dhcp" {
		return fmt.Errorf("device mode only support static or dhcp, mode %s invalid", mode)
	}
	// 静态模式下，核心参数不能为空
	if mode == "static" {
		if ip == "" || mask == "" || gateway == "" {
			return errors.New("static config, ip/mask/gateway can not be empty")
		}
		// 校验子网掩码格式
		if _, err := strconv.Atoi(subnetMaskToPrefix(mask)); err != nil {
			return fmt.Errorf("invalid subnet mask: %s, err: %v", mask, err)
		}
	}

	// 步骤2：根据设备名查找对应的连接名
	deviceConnOutput, err := execCommand("nmcli", "-t", "-f", "DEVICE,CONNECTION", "device", "status")
	if err != nil {
		return fmt.Errorf("get device status failed: %v", err)
	}
	var connName string
	deviceLines := strings.Split(deviceConnOutput, "\n")
	for _, line := range deviceLines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		if strings.TrimSpace(parts[0]) == deviceName {
			connName = strings.TrimSpace(parts[1])
			break
		}
	}
	if connName == "" || connName == "--" {
		return fmt.Errorf("device [%s] can not find valid connection", deviceName)
	}

	// 步骤3：拼接 nmcli 修改命令
	modifyCmd := []string{"connection", "modify", connName}

	// 配置地址模式 + 对应参数
	if mode == "static" {
		// 静态模式：转换 IP+掩码 为 CIDR 格式
		cidr := fmt.Sprintf("%s/%s", ip, subnetMaskToPrefix(mask))
		modifyCmd = append(modifyCmd,
			"ipv4.method", "manual",
			"ipv4.addresses", cidr,
			"ipv4.gateway", gateway,
		)
	} else {
		// DHCP 模式：清空静态配置，设置自动获取
		modifyCmd = append(modifyCmd,
			"ipv4.method", "auto",
			"ipv4.addresses", "",
			"ipv4.gateway", "",
		)
	}

	// 配置 DNS（主+备用，空格分隔）
	dnsStr := ""
	if dns != "" {
		dnsStr = dns
		if backupDns != "" {
			dnsStr += " " + backupDns
		}
	}
	modifyCmd = append(modifyCmd, "ipv4.dns", dnsStr)

	// 步骤4：执行修改命令（需 root 权限）
	fullArgs := append([]string{"nmcli"}, modifyCmd...)
	_, err = execCommand("sudo", fullArgs...)
	//_, err = execCommand("sudo", "nmcli", modifyCmd...)
	if err != nil {
		return fmt.Errorf("modify network failed: %v", err)
	}

	// 步骤5：重新激活连接，使配置生效
	_, err = execCommand("sudo", "nmcli", "connection", "up", connName)
	if err != nil {
		return fmt.Errorf("nmcli connection up failed: %v", err)
	}

	fmt.Printf("device[%s] network config success, current mode: %s\n", deviceName, mode)
	return nil
}
