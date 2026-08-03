package emsclient

import (
	"webserver-arm/models"

	"github.com/goccy/go-json"
)

const (
	// ststem call 500-699
	QueryProcMethod  = 501
	UpdateProcMethod = 502
	AccessSnMethod   = 503
)

// queryyx
//
//nolint:govet
type AccessSnReq struct {
	// 1:get  2:update
	Op     int    `json:"op"`
	Sn     string `json:"sn"`
	Addr   string `json:"addr"`
	Name   string `json:"name"`
	CabNum int    `json:"cabnum"`
}

func (e *AccessSnReq) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func (e *AccessSnReq) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

type SnData struct {
	Sn     string `json:"sn"`
	Name   string `json:"name"`
	Addr   string `json:"addr"`
	CabNum int    `json:"cabnum"`
}

type CommonSnData struct {
	Topic   string `json:"topic"`
	Content SnData `json:"content"`
}

//nolint:govet
type ProjSnResp struct {
	Code    int          `json:"code"`
	Message string       `json:"msg"`
	Data    CommonSnData `json:"data"`
}

func (e *ProjSnResp) Clone() models.IEmsRPCResp {
	return &ProjSnResp{e.Code, e.Message, e.Data}
}

func (e *ProjSnResp) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
func (e *ProjSnResp) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *EmsClient) AccessSn(req *AccessSnReq) (resp *ProjSnResp, err error) {
	method := uint32(AccessSnMethod)
	resp = &ProjSnResp{}
	err = e.Call(method, req, resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}
