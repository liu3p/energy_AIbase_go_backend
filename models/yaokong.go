package models

import "github.com/goccy/go-json"

type Yaokong struct {
	Data string
}

type RemoteCallReq struct {
	DevSn  string `json:"devsn"`
	Method string `json:"method"`
	Seq    int64  `json:"seq"`
	Value  int    `json:"value"`
}

func (e *RemoteCallReq) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
func (e *RemoteCallReq) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}
