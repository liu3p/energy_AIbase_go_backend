package models

//nolint:tagliatelle
type SysProcReq struct {
	ProcName string `json:"ProcName"`
	Action   string `json:"Action"`
	ProcPid  int    `json:"ProcPid"`
}

//nolint:tagliatelle
type SysProcResp struct {
	Procs []SysProcRespEntry `json:"Procs"`
}

//nolint:tagliatelle
type SysProcRespEntry struct {
	ProcDesc    string  `json:"ProcDesc"`
	ProcName    string  `json:"ProcName"`
	RunStatus   string  `json:"RunStatus"`
	ProcWarning string  `json:"ProcWarning"`
	ProcPid     int     `json:"ProcPid"`
	ProcCPU     float32 `json:"ProcCpu"`
	ProcMem     float32 `json:"ProcMem"`
}
