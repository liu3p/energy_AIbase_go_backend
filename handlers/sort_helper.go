package handlers

import (
	"sort"
	"webserver-arm/emsxmldb"
)

// priorityParamName 优先排在最前的参数名（完整名称，含空格）
const priorityParamName = "STATION_ID ASDU_ADDR"

// parameterLess Parameters 排序比较函数：priorityParamName 排最前，其余按字母序
func parameterLess(a, b string) bool {
	if a == priorityParamName {
		return true
	}
	if b == priorityParamName {
		return false
	}
	return a < b
}

// sortParameters 拷贝 Parameters 切片并排序，不修改原始数据
// 排序规则：STATION_ID ASDU_ADDR 排最前，其余按字母序
func sortParameters(params []emsxmldb.Parameter) []emsxmldb.Parameter {
	cp := make([]emsxmldb.Parameter, len(params))
	copy(cp, params)
	sort.Slice(cp, func(i, j int) bool {
		return parameterLess(cp[i].Name, cp[j].Name)
	})
	return cp
}

// sortChannelPlugins 拷贝 Plugins 切片并对每个 Plugin 的 Parameters 排序
func sortChannelPlugins(plugins []emsxmldb.Plugin) []emsxmldb.Plugin {
	cp := make([]emsxmldb.Plugin, len(plugins))
	for i, p := range plugins {
		cp[i] = p
		cp[i].Parameters = sortParameters(p.Parameters)
	}
	return cp
}

// sortChannelParameters 返回 Channel 的浅拷贝副本，其中每个 Plugin 的 Parameters 已排序
// STATION_ID ASDU_ADDR 排最前，其余按字母序
func sortChannelParameters(ch *emsxmldb.Channel) *emsxmldb.Channel {
	if ch == nil {
		return nil
	}
	cp := *ch
	cp.Plugins = sortChannelPlugins(ch.Plugins)
	return &cp
}

// sortChannelGroupPlugins 对 ChannelGroup 内所有 Channel 的 Plugins.Parameters 排序（拷贝）
func sortChannelGroupPlugins(cg *emsxmldb.ChannelGroup) *emsxmldb.ChannelGroup {
	if cg == nil {
		return nil
	}
	cp := *cg
	cp.Channels = make([]emsxmldb.Channel, len(cg.Channels))
	for i, ch := range cg.Channels {
		cp.Channels[i] = ch
		cp.Channels[i].Plugins = sortChannelPlugins(ch.Plugins)
	}
	return &cp
}
