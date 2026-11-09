// strategyprobe 打印策略动态库中的参数信息。
// 入参为动态库名（完整路径 / 文件名 / 关键字），通过 cgolibparse.GetStrategyLibInfo
// 调用底层 agc 基础库的 agc_get_strategy_lib_info 接口解析出策略信息。
//
// 用法（需在具备 agc 基础库与头文件的 ARM 目标机上构建运行）：
//
//	go build -o strategyprobe ./cmd/strategyprobe
//	./strategyprobe libdemo.so
//	./strategyprobe demo            # 关键字模糊匹配
//	./strategyprobe -json libdemo.so
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"webserver-arm/cgolibparse"
	"webserver-arm/log"
)

func main() {
	jsonOut := flag.Bool("json", false, "以原始 JSON 形式输出完整策略信息")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法: %s [选项] <动态库名>\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintln(os.Stderr, "动态库名可以是:")
		fmt.Fprintln(os.Stderr, "  完整路径  如 /root/arm/lib/strategy/libdemo.so")
		fmt.Fprintln(os.Stderr, "  文件名    如 libdemo.so (自动到 $ARM_SF/lib/strategy 下查找)")
		fmt.Fprintln(os.Stderr, "  关键字    如 demo        (模糊匹配 $ARM_SF/lib/strategy 下的库)")
		fmt.Fprintln(os.Stderr)
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	libPath, err := resolveLibPath(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	// cgolibparse 内部会用到 log，确保 logger 已初始化
	log.InitLog()

	info, err := cgolibparse.GetStrategyLibInfo(libPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "解析动态库失败: %v\n", err)
		os.Exit(1)
	}

	if *jsonOut {
		b, err := json.MarshalIndent(info, "", "    ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "序列化失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(b))
		return
	}

	printInfo(libPath, info)
}

// resolveLibPath 把用户输入解析为磁盘上唯一确定的一个动态库文件。
func resolveLibPath(name string) (string, error) {
	// 1) 直接可用（完整路径或当前目录下的文件）
	if fi, err := os.Stat(name); err == nil && !fi.IsDir() {
		return name, nil
	}

	base := os.Getenv("ARM_SF")
	if base == "" {
		return "", fmt.Errorf("环境变量 ARM_SF 未设置，且 %q 不是有效文件路径", name)
	}
	libDir := filepath.Join(base, "lib", "strategy")

	// 2) 目录 + 文件名
	candidate := filepath.Join(libDir, name)
	if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
		return candidate, nil
	}

	// 3) 关键字模糊匹配
	matches, _ := filepath.Glob(filepath.Join(libDir, "*"+name+"*"))
	if len(matches) == 0 {
		matches, _ = filepath.Glob(filepath.Join(libDir, "lib"+name+"*"))
	}

	switch len(matches) {
	case 0:
		avail, _ := filepath.Glob(filepath.Join(libDir, "lib*.so*"))
		sort.Strings(avail)
		if len(avail) == 0 {
			return "", fmt.Errorf("在 %s 下未找到任何策略库", libDir)
		}
		return "", fmt.Errorf("未找到匹配 %q 的动态库，%s 下可用的库:\n  %s",
			name, libDir, strings.Join(avail, "\n  "))
	case 1:
		return matches[0], nil
	default:
		sort.Strings(matches)
		return "", fmt.Errorf("%q 匹配到多个动态库，请指定更精确的名字:\n  %s",
			name, strings.Join(matches, "\n  "))
	}
}

func printInfo(libPath string, info *cgolibparse.StrategyLibInfo) {
	fmt.Printf("库文件:   %s\n", libPath)
	fmt.Printf("策略名称: %s\n", info.Name)
	fmt.Printf("描述:     %s\n", info.Desc)
	fmt.Printf("优先级:   %d\n", info.Priority)

	printSettingTable("Setting (配置参数)", info.Setting)
	printSettingTable("CalcPara (计算参数)", info.CalcPara)

	fmt.Printf("\n--- DynPara (动态参数): %d 项 ---\n", len(info.DynPara))
	if len(info.DynPara) == 0 {
		fmt.Println("(无)")
		return
	}
	for i, d := range info.DynPara {
		fmt.Printf("  %2d. %s\n", i+1, d)
	}
}

func printSettingTable(title string, items []cgolibparse.SettingItem) {
	fmt.Printf("\n--- %s: %d 项 ---\n", title, len(items))
	if len(items) == 0 {
		fmt.Println("(无)")
		return
	}
	fmt.Printf("%-4s %-28s %-10s %-14s %-10s %-10s %s\n",
		"序号", "名称", "类型", "值", "最小", "最大", "描述")
	for i, it := range items {
		fmt.Printf("%-4d %-28s %-10s %-14s %-10s %-10s %s\n",
			i+1, it.Name, it.Type, it.Value, it.Min, it.Max, it.Desc)
	}
}
