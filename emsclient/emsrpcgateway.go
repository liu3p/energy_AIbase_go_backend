package emsclient

import (
	"webserver-arm/models"

	"github.com/goccy/go-json"
)

const (
	// gateways  200-499
	QueryDevMethod    = 200
	QueryYxMethod     = 201
	QueryYcMethod     = 202
	QueryKwhMethod    = 203
	QueryPropMethod   = 204
	QueryRemoteMethod = 205
)

// queryyx
type QueryYXReq []string

func (e *QueryYXReq) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func (e *QueryYXReq) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *EmsClient) QueryYX(req *QueryYXReq) (resp *models.QueryTeleResp, err error) {
	method := uint32(QueryYxMethod)
	resp = &models.QueryTeleResp{}
	err = e.Call(method, req, resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// queryyc
type QueryYCReq []string

func (e *QueryYCReq) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func (e *QueryYCReq) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *EmsClient) QueryYC(req *QueryYCReq) (resp *models.QueryTeleResp, err error) {
	method := uint32(QueryYcMethod)
	resp = &models.QueryTeleResp{}
	err = e.Call(method, req, resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// queryprop
type QueryPropReq []string

func (e *QueryPropReq) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func (e *QueryPropReq) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *EmsClient) QueryProp(req *QueryPropReq) (resp *models.QueryPropResp, err error) {
	method := uint32(QueryPropMethod)
	resp = &models.QueryPropResp{}
	err = e.Call(method, req, resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// query devices

func (e *EmsClient) QueryDevices() (resp *models.QueryDevicesResp, err error) {
	method := uint32(QueryDevMethod)
	resp = &models.QueryDevicesResp{}
	err = e.Call(method, &models.CommonEmptyReq{}, resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// remote send
func (e *EmsClient) RemoteDevices(req *models.RemoteCallReq) (resp *models.CommonTransResp, err error) {
	method := uint32(QueryRemoteMethod)
	resp = &models.CommonTransResp{}
	err = e.Call(method, req, resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}
