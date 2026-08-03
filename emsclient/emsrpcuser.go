package emsclient

import (
	"webserver-arm/models"

	"github.com/goccy/go-json"
)

const (
	// users  101-199
	LoginMethod     = 101
	LogoutMethod    = 102
	UpdatePwdMethod = 103
	ResetPwdMethod  = 104
)

// login
type LoginReq struct {
	UserName string `json:"uname"`
	UserPwd  string `json:"upwd"`
}

func (e *LoginReq) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
func (e *LoginReq) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

//nolint:govet
type LoginResp struct {
	Result   int    `json:"result"`
	ErrInfo  string `json:"errinfo"`
	UserType string `json:"utype"`
}

func (e *LoginResp) Clone() models.IEmsRPCResp {
	return &LoginResp{e.Result, e.ErrInfo, e.UserType}
}

func (e *LoginResp) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
func (e *LoginResp) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *EmsClient) Login(req *LoginReq) (resp *LoginResp, err error) {
	method := uint32(LoginMethod)
	resp = &LoginResp{}
	err = e.Call(method, req, resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// update pwd
type UpdatePwdReq struct {
	UserName string `json:"uname"`
	NewPwd   string `json:"newpwd"`
	OldPwd   string `json:"oldpwd"`
}

func (e *UpdatePwdReq) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
func (e *UpdatePwdReq) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

//nolint:govet
type UpdatePwdResp struct {
	Result  int    `json:"result"`
	ErrInfo string `json:"errinfo"`
}

func (e *UpdatePwdResp) Clone() models.IEmsRPCResp {
	return &UpdatePwdResp{e.Result, e.ErrInfo}
}

func (e *UpdatePwdResp) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
func (e *UpdatePwdResp) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *EmsClient) UpdatePwd(req *UpdatePwdReq) (resp *UpdatePwdResp, err error) {
	method := uint32(UpdatePwdMethod)
	resp = &UpdatePwdResp{}
	err = e.Call(method, req, resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// reset pwd
type ResetPwdReq struct {
	UserName string `json:"uname"`
	Pwd      string `json:"pwd"`
}

func (e *ResetPwdReq) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
func (e *ResetPwdReq) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

//nolint:govet
type ResetPwdResp struct {
	Result  int    `json:"result"`
	ErrInfo string `json:"errinfo"`
}

func (e *ResetPwdResp) Clone() models.IEmsRPCResp {
	return &ResetPwdResp{e.Result, e.ErrInfo}
}

func (e *ResetPwdResp) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
func (e *ResetPwdResp) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *EmsClient) ResetPwd(req *ResetPwdReq) (resp *ResetPwdResp, err error) {
	method := uint32(ResetPwdMethod)
	resp = &ResetPwdResp{}
	err = e.Call(method, req, resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}
