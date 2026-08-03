package models

import "github.com/goccy/go-json"

type DeviceInfo struct {
	Sn      string `json:"sn"`
	Name    string `json:"name"`
	DevCode string `json:"dev_code"`
	Gateway string `json:"gateway"`
	Status  int    `json:"status"` // 1 - offline, 2 - online
}

//nolint:govet
type DeviceData struct {
	Topic   string                `json:"topic"`
	Content map[string]DeviceInfo `json:"content"`
}

//nolint:govet
type QueryDevicesResp struct {
	Code    int        `json:"code"`
	Data    DeviceData `json:"data"`
	Message string     `json:"msg"`
}

func (e *QueryDevicesResp) Clone() IEmsRPCResp {
	return &QueryDevicesResp{Code: e.Code, Message: e.Message, Data: e.Data}
}

func (e *QueryDevicesResp) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
func (e *QueryDevicesResp) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

type KeyValueInfo struct {
	K  string  `json:"k"`
	V  float32 `json:"v"`
	At int64   `json:"at,omitempty"`
	Mt int64   `json:"mt,omitempty"`
}

//nolint:govet
type TeleData struct {
	Topic   string                    `json:"topic"`
	Content map[string][]KeyValueInfo `json:"content"`
}

//nolint:govet
type QueryTeleResp struct {
	Code    int      `json:"code"`
	Message string   `json:"msg"`
	Data    TeleData `json:"data"`
}

func (e *QueryTeleResp) Clone() IEmsRPCResp {
	return &QueryTeleResp{Code: e.Code, Message: e.Message, Data: e.Data}
}

func (e *QueryTeleResp) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
func (e *QueryTeleResp) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

type KeyValueInfoEx struct {
	K  string      `json:"k"`
	V  interface{} `json:"v"`
	At int64       `json:"at,omitempty"`
	Mt int64       `json:"mt,omitempty"`
}

//nolint:govet
type PropData struct {
	Topic   string                      `json:"topic"`
	Content map[string][]KeyValueInfoEx `json:"content"`
}

//nolint:govet
type QueryPropResp struct {
	Code    int      `json:"code"`
	Message string   `json:"msg"`
	Data    PropData `json:"data"`
}

func (e *QueryPropResp) Clone() IEmsRPCResp {
	return &QueryPropResp{Code: e.Code, Message: e.Message, Data: e.Data}
}

func (e *QueryPropResp) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
func (e *QueryPropResp) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}
