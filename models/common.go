package models

import (
	"github.com/goccy/go-json"
)

type IEmsRPCResp interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
	Clone() IEmsRPCResp
}

//nolint:govet
type CommonResp struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

//nolint:govet
type CommonData struct {
	Topic   string `json:"topic"`
	Content any    `json:"content"`
}

func (e *CommonData) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
func (e *CommonData) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

//nolint:govet
type CommonTransResp struct {
	Code    int        `json:"code"`
	Data    CommonData `json:"data"`
	Message string     `json:"msg"`
}

func (e *CommonTransResp) Clone() IEmsRPCResp {
	return &CommonTransResp{Code: e.Code, Message: e.Message, Data: e.Data}
}

func (e *CommonTransResp) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
func (e *CommonTransResp) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

//nolint:govet
type QueryI18nResp struct {
	Version string                       `json:"version"`
	Content map[string]map[string]string `json:"content"`
}

//nolint:govet
type UnitData struct {
	Unit   string            `json:"unit"`
	Values map[string]string `json:"values"`
}

//nolint:govet
type QueryDataUnitResp struct {
	Version string                 `json:"version"`
	Content map[string]interface{} `json:"content"`
}

//nolint:govet
type QueryDataUnit struct {
	Version string              `json:"version"`
	Content map[string]UnitData `json:"content"`
}

type CommonEmptyReq struct{}

func (e *CommonEmptyReq) Marshal() ([]byte, error) {
	return []byte{}, nil
}
func (e *CommonEmptyReq) Unmarshal(data []byte) error {
	return nil
}

type GatewayInfo struct {
	Sn     string `json:"sn"`
	Name   string `json:"name"`
	Status int    `json:"status"`
}
