package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"time"
	"webserver-arm/emsclient"
	"webserver-arm/handlers"
	"webserver-arm/models"

	"github.com/goccy/go-json"

	"github.com/gorilla/websocket"
)

var token = ""

var userName = "admin"   //"oper-3"
var userPwd = "superman" //"snr"

var address = "192.168.2.27:38080" //"127.0.0.1" //"192.168.1.104" //
var http_url = "http://" + address
var ws_url = "ws://" + address

func login() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	// 创建一个GET请求
	logData := &handlers.LoginReq{Uname: userName, Pwd: userPwd}
	data, err := json.Marshal(logData)
	if err != nil {
		fmt.Println("Marshal failed")
		return err
	}

	buf := bytes.NewReader(data)
	req, err := http.NewRequest("POST", http_url+"/api/v1/user/login", buf)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("do login failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	loginResp := &handlers.LoginResp{}
	err = json.Unmarshal(dataInfo, loginResp)
	if err != nil {
		fmt.Println("Unmarshal failed:", err)
		return err
	}

	fmt.Println("Login Success:", loginResp)
	token = loginResp.AccessToken
	return nil
}

func queryYX() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	// 创建一个GET请求
	qyxData := make([]string, 0)
	qyxData = append(qyxData, "esscab01ac1")
	data, err := json.Marshal(qyxData)
	if err != nil {
		fmt.Println("Marshal failed")
		return err
	}

	buf := bytes.NewReader(data)
	req, err := http.NewRequest("POST", http_url+"/api/v1/log/gateways/esscab1/dev_telesignal", buf)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("do login failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("query yx:", string(dataInfo))
	return nil
}

func queryYX2() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	// 创建一个GET请求
	req, err := http.NewRequest("GET", http_url+"/api/v1/log/gateways/esscab1/devices/esscab1pcs1/telesignal", nil)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("do login failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("query yx2:", string(dataInfo))
	return nil
}

func querySN() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	req, err := http.NewRequest("GET", http_url+"/api/v1/log/system/project/sn", nil)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("do login failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("query sn:", string(dataInfo))
	return nil
}

func queryGW() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	req, err := http.NewRequest("GET", http_url+"/api/v1/log/gateways", nil)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("do login failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryGW:", string(dataInfo))
	return nil
}

func queryModbusTransTab() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	req, err := http.NewRequest("GET", http_url+"/api/v1/log/system/modbus/transdata", nil)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("do login failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryGW:", string(dataInfo))
	return nil
}

func queryModbusTab() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	req, err := http.NewRequest("GET", http_url+"/api/v1/log/system/modbus/config", nil)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("do login failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryGW:", string(dataInfo))
	return nil
}

func queryI18n() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	req, err := http.NewRequest("GET", http_url+"/api/v1/log/system/i18n?current_version=1", nil)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("do login failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryI18n:", string(dataInfo), ",code:", resp.StatusCode)
	return nil
}

func queryRedis() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	// 创建一个GET请求
	rdData := &handlers.RdData{}
	rdData.Content = "sasdbasbdasbdklsad"
	data, err := json.Marshal(rdData)
	if err != nil {
		fmt.Println("Marshal failed")
		return err
	}

	buf := bytes.NewReader(data)
	req, err := http.NewRequest("POST", http_url+"/api/v1/log/rd_query/123", buf)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("do login failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("query redis:", string(dataInfo))
	return nil
}

func queryDev() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	req, err := http.NewRequest("GET", http_url+"/api/v1/log/gateways/esscab1/devices", nil)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("do login failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryDev:", string(dataInfo))
	return nil
}

func queryEms() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	req, err := http.NewRequest("GET", http_url+"/api/v1/log/ems", nil)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("do login failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("queryEms ReadAll:", err)
		return err
	}

	fmt.Println("queryEms:", string(dataInfo))
	return nil
}

func queryEms2() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	req, err := http.NewRequest("GET", http_url+"/api/v1/log/ems/", nil)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("do login failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("queryEms ReadAll:", err)
		return err
	}

	fmt.Println("queryEms:", string(dataInfo))
	return nil
}

func queryEmsYX() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	req, err := http.NewRequest("GET", http_url+"/api/v1/log/ems/telesignal", nil)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("do login failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryDev:", string(dataInfo))
	return nil
}

func queryEmsYC() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	req, err := http.NewRequest("GET", http_url+"/api/v1/log/ems/telemetry", nil)
	//req, err := http.NewRequest("GET", http_url+"/api/v1/log/ems", nil)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("do login failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryDev:", string(dataInfo))
	return nil
}

func queryProjSn() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	//req, err := http.NewRequest("GET", "http://127.0.0.1/api/v1/log/ems/telemetry", nil)
	req, err := http.NewRequest("GET", http_url+"/api/v1/log/system/project/sn", nil)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("do login failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryProjSn:", string(dataInfo))
	return nil
}

func queryProp() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	//req, err := http.NewRequest("GET", "http://127.0.0.1/api/v1/log/gateways/wesscab01/devices/esscab01pcs1/prop", nil)
	req, err := http.NewRequest("GET", http_url+"/api/v1/log/ems/prop", nil)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("do login failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryProp:", string(dataInfo))
	return nil
}

func updateProjSn() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	prjectSn := &emsclient.SnData{Sn: "98765432", Name: "27测试机", Addr: "江苏南京分公司", CabNum: 2}
	data, err := json.Marshal(prjectSn)
	if err != nil {
		fmt.Println("Marshal failed")
		return err
	}

	buf := bytes.NewReader(data)
	//req, err := http.NewRequest("GET", "http://127.0.0.1/api/v1/log/ems/telemetry", nil)
	req, err := http.NewRequest("POST", http_url+"/api/v1/log/system/project/sn", buf)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("do login failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("updateProjSn:", string(dataInfo))
	return nil
}

func queryEmsRemote() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	// 创建一个GET请求
	rdData := &handlers.RemoteReq{}
	rdData.Content = "[{\"type\":1,\"bus\":\"bus1\",\"date\":\"2006-01-02\",\"config_data\":{\"plans\":[{\"p\":100.45,\"bt\":\"00:00\",\"et\":\"01:21\"},{\"p\":0,\"bt\":\"23:21\",\"et\":\"24:00\"},{\"p\":-100.45,\"bt\":\"01:21\",\"et\":\"23:21\"}],\"soc_up\":100,\"soc_dn\":48.5,\"interval\":15}},{\"type\":1,\"bus\":\"bus2\",\"date\":\"2006-01-02\",\"config_data\":{\"plans\":[{\"p\":110.45,\"bt\":\"00:00\",\"et\":\"01:21\"},{\"p\":10,\"bt\":\"23:21\",\"et\":\"24:00\"},{\"p\":-100.45,\"bt\":\"01:21\",\"et\":\"23:21\"}],\"soc_up\":100,\"soc_dn\":28,\"interval\":15}}]"
	data, err := json.Marshal(rdData)
	if err != nil {
		fmt.Println("Marshal failed")
		return err
	}

	buf := bytes.NewReader(data)
	req, err := http.NewRequest("POST", http_url+"/api/v1/log/ems/remote/PolicyConfig", buf)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("queryEmsRemote failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryEmsRemote:", string(dataInfo))
	return nil
}

func queryDevRemote() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	// 创建一个GET请求
	rdData := &handlers.RemoteReq{}
	rdData.Content = "{\"state\":2}"
	rdData.ProjSn = "98765432"
	data, err := json.Marshal(rdData)
	if err != nil {
		fmt.Println("Marshal failed")
		return err
	}

	buf := bytes.NewReader(data)
	req, err := http.NewRequest("POST", http_url+"/api/v1/log/gateways/esscab01/devices/esscab01pcs1/remote/RunModeSet", buf)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("do login failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryDevRemote:", string(dataInfo))
	return nil
}

func getAllOnlines() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	// 创建一个GET请求
	req, err := http.NewRequest("GET", http_url+"/api/v1/log/user/onlines", nil)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("getAllOnlines failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("getAllOnlines:", string(dataInfo))
	return nil
}

func queryEmsRemote2() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	// 创建一个GET请求
	rdData := &handlers.RemoteReq{}
	rdData.ProjSn = "11223344"
	dataStr := "[{\"mode\":0,\"bus\":\"bus1\",\"date\":\"2006-01-02\",\"p\":100.0,\"max_p\":322.1,\"soc_up\":85,\"soc_dn\":15,\"chg_max\":100,\"dchg_max\":200,\"chg_eff\":80,\"dchg_eff\":90}]"
	json.Unmarshal([]byte(dataStr), &rdData.Content)
	data, err := json.Marshal(rdData)
	if err != nil {
		fmt.Println("Marshal failed")
		return err
	}

	buf := bytes.NewReader(data)
	req, err := http.NewRequest("POST", http_url+"/api/v1/log/ems/remote/PlanSet", buf)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("queryEmsRemote2 failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryEmsRemote2:", string(dataInfo))
	return nil
}

func queryEmsRemote3() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	// 创建一个GET请求
	rdData := &handlers.RemoteReq{}
	rdData.Content = "[{\"max_p\":233.2,\"bus\":\"bus1\",\"type\":1,\"status\":0,\"from\":2,\"exec_t\":\"2023-09-15 14:17:00\",\"chg_max\":40.2,\"dchg_max\":40.2,\"chg_eff\":89.9,\"dchg_eff\":89.9,\"ele_state\":0,\"chg_cpt\":100,\"dchg_cpt\":20,\"soc\":25,\"soc_up\":85,\"soc_dn\":15}]"

	data, err := json.Marshal(rdData)
	if err != nil {
		fmt.Println("Marshal failed")
		return err
	}

	buf := bytes.NewReader(data)
	req, err := http.NewRequest("POST", http_url+"/api/v1/log/ems/remote/LocalPolicyUpSmart", buf)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("queryEmsRemote3 failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryEmsRemote3:", string(dataInfo))
	return nil
}

func queryEmsRemote4() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	// 创建一个GET请求
	rdData := &handlers.RemoteReq{}
	dataStr := "[{\"max_p\":233.2,\"bus\":\"bus1\",\"type\":1,\"status\":0,\"from\":2,\"exec_t\":\"2023-09-15 14:17:00\",\"chg_max\":40.2,\"dchg_max\":40.2,\"chg_eff\":89.9,\"dchg_eff\":89.9,\"ele_state\":0,\"chg_cpt\":100,\"dchg_cpt\":20,\"soc\":25,\"soc_up\":85,\"soc_dn\":15}]"
	json.Unmarshal([]byte(dataStr), &rdData.Content)
	rdData.ProjSn = "11223344"
	data, err := json.Marshal(rdData)
	if err != nil {
		fmt.Println("Marshal failed")
		return err
	}

	buf := bytes.NewReader(data)
	req, err := http.NewRequest("POST", http_url+"/api/v1/log/ems/remote/LocalPolicyUpSmart", buf)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("queryEmsRemote4 failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryEmsRemote4:", string(dataInfo))
	return nil
}

func queryEmsRemote5() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	// 创建一个GET请求
	rdData := &handlers.RemoteReq{}
	rdData.Content = "[{\"type\":1,\"bus\":\"bus1\",\"status\":0,\"exec_t\":\"2023-09-15 14:17:00\",\"soc_up\":90,\"soc_dn\":15,\"p\":100.1,\"from\":0}]"

	data, err := json.Marshal(rdData)
	if err != nil {
		fmt.Println("Marshal failed")
		return err
	}

	buf := bytes.NewReader(data)
	req, err := http.NewRequest("POST", http_url+"/api/v1/log/ems/remote/LocalPolicyUpHand", buf)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("queryEmsRemote5 failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryEmsRemote5:", string(dataInfo))
	return nil
}

func queryEmsRemote6() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	// 创建一个GET请求
	rdData := &handlers.RemoteReq{}
	rdData.Content = "{\"state:\": \"0\"}"

	data, err := json.Marshal(rdData)
	if err != nil {
		fmt.Println("Marshal failed")
		return err
	}

	buf := bytes.NewReader(data)
	req, err := http.NewRequest("POST", http_url+"/api/v1/log/ems/remote/AGCExitCtrl", buf)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("queryEmsRemote5 failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryEmsRemote5:", string(dataInfo))
	return nil
}

func queryOnlines() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	// 创建一个GET请求
	req, err := http.NewRequest("GET", http_url+"/api/v1/log/user/onlines", nil)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("queryOnlines failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryOnlines:", string(dataInfo))
	return nil
}

func kickOnlines() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	// 创建一个POST请求
	kickData := "{\"username\": \"oper-3\"}"

	buf := bytes.NewBuffer([]byte(kickData))
	req, err := http.NewRequest("POST", http_url+"/api/v1/log/user/kickout", buf)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("kickOnlines failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("kickOnlines:", string(dataInfo))
	return nil
}

func queryMqttIdHis() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	// 创建一个POST请求
	hisData := "{\"date\": \"2024-01-26\"}"
	buf := bytes.NewBuffer([]byte(hisData))
	req, err := http.NewRequest("POST", http_url+"/api/v1/log/gateways/ems/devices/ems/history/AgcSoh", buf)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("queryMqttIdHis failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryMqttIdHis:", string(dataInfo))
	return nil
}

func queryLogDumpStart() (error, int64) {
	// 创建一个HTTP客户端
	client := &http.Client{}

	// 创建一个POST请求
	hisData := "{\"operation\": 1, \"gatewaysn\":\"ems\"}"
	buf := bytes.NewBuffer([]byte(hisData))
	req, err := http.NewRequest("POST", http_url+"/api/v1/log/rd/logdump", buf)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err, 0
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("queryLogDumpStart failed:", err)
		return err, 0
	}

	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err, 0
	}

	dataInfo := make(map[string]int64)
	err = json.Unmarshal(bodyData, &dataInfo)
	if err != nil {
		fmt.Println("Unmarshal body err:", err, ", bodyData:", string(bodyData))
		return err, 0
	}
	fmt.Println("queryLogDumpStart:", dataInfo)
	return nil, dataInfo["taskid"]
}

func queryLogDumpStop(taskid int64) (error, int64) {
	// 创建一个HTTP客户端
	client := &http.Client{}

	// 创建一个POST请求
	hisData := "{\"operation\": 2, \"taskid\": " + strconv.FormatInt(taskid, 10) + ", \"gatewaysn\":\"ems\"}"
	buf := bytes.NewBuffer([]byte(hisData))
	req, err := http.NewRequest("POST", http_url+"/api/v1/log/rd/logdump", buf)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err, 0
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("queryLogDumpStop failed:", err)
		return err, 0
	}

	bodyData, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err, 0
	}

	dataInfo := make(map[string]int64)
	err = json.Unmarshal(bodyData, &dataInfo)
	if err != nil {
		fmt.Println("Unmarshal body err:", err, ", bodyData:", string(bodyData))
		return err, 0
	}
	fmt.Println("queryLogDumpStop:", dataInfo)
	return nil, dataInfo["taskid"]
}

func queryGetLogDump(id int64) error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	// 创建一个GET请求
	// hisData := "{\"duration\": 10, \"gatewaysn\":\"ems\"}"
	// buf := bytes.NewBuffer([]byte(hisData))
	req, err := http.NewRequest("GET", http_url+"/api/v1/log/rd/logdump/"+strconv.FormatInt(id, 10), nil)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("queryGetLogDump failed:", err)
		return err
	}

	contentDisposition := resp.Header.Get("Content-Disposition")
	fileName := extractFileName(contentDisposition)
	fmt.Println("extractFileName:", fileName, ",resp header:", resp.Header)
	if fileName == "" {
		fileName = strconv.FormatInt(id, 10) + ".tgz"
	}
	localFile, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Create file ", fileName, " failed:", err)
		return err
	}
	defer localFile.Close()

	_, err = io.Copy(localFile, resp.Body)
	if err != nil {
		fmt.Println("Copy Error:", err)
		return err
	}

	fmt.Println("queryGetLogDump over:", localFile.Name())
	return nil
}

// 从Content-Disposition头中提取文件名
func extractFileName(contentDisposition string) string {
	re := regexp.MustCompile(`filename="(.+)"`)
	matches := re.FindStringSubmatch(contentDisposition)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

func queryProclistGet() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	// 创建一个POST请求

	req, err := http.NewRequest("GET", http_url+"/api/v1/log/rd/proclist/esscab1", nil)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("queryProclistGet failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryProclistGet:", string(dataInfo))
	return nil
}

func queryProclistPost() error {
	// 创建一个HTTP客户端
	client := &http.Client{}

	// 创建一个POST请求
	reqInfo := &models.SysProcReq{
		ProcPid:  23804,
		ProcName: "rdbmon",
		Action:   "Kill", // Logout,Kill, Start, Restart, RestartBus
	}
	body, err := json.Marshal(reqInfo)
	if err != nil {
		fmt.Println("Marshal SysProcReq failed:", err)
		return err
	}

	req, err := http.NewRequest("POST", http_url+"/api/v1/log/rd/proclist/esscab1", bytes.NewBuffer(body))
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("queryProclistPost failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryProclistPost:", string(dataInfo))
	return nil
}

func queryIPConfig() error {
	// 创建一个HTTP客户端
	client := &http.Client{}
	req, err := http.NewRequest("GET", http_url+"/api/v1/log/system/network/config", nil)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)
	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("queryIPConfig failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryIPConfig:", string(dataInfo))
	return nil

}

func queryLocalProcs() error {
	// 创建一个HTTP客户端
	client := &http.Client{}
	req, err := http.NewRequest("GET", http_url+"/api/v1/log/system/process", nil)
	if err != nil {
		fmt.Println("NewRequest failed:", err)
		return err
	}

	req.Header.Add("Authorization", token)
	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("queryLocalProcs failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryLocalProcs:", string(dataInfo))
	return nil

}

func queryLogTypes() error {
	pid := 0
	fmt.Print("Input queryLogTypes pid:")
	fmt.Scan(&pid)
	serverURL := fmt.Sprintf("%s/api/v1/log/system/process/%d/logtype", http_url, pid)
	client := &http.Client{}
	req, err := http.NewRequest("GET", serverURL, nil)
	if err != nil {
		fmt.Println("queryLogTypes NewRequest failed:", err)
		return err
	}
	req.Header.Add("Authorization", token)
	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("queryLogTypes failed:", err)
		return err
	}

	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("queryLogTypes:", string(dataInfo))
	return nil
}

func queryLogs() error {
	pid := 0
	logType := 0
	fmt.Print("Input queryLogs pid:")
	fmt.Scan(&pid)
	fmt.Print("Input queryLogs logtype:")
	fmt.Scan(&logType)
	utlstr := url.QueryEscape(token)
	serverURL := fmt.Sprintf("%s/api/v1/log/system/process/%d/logdump/%d?authorization=%s", ws_url, pid, logType, utlstr)
	u, err := url.Parse(serverURL)
	if err != nil {
		fmt.Println("Invalid server URL:", err)
		return err
	}
	fmt.Println("server push url:", u.String())
	header := http.Header{}
	header.Add("Authorization", token)

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Println("WebSocket connection failed:", err)
		return err
	}
	defer conn.Close()
	conn.WriteMessage(websocket.TextMessage, []byte("queryLOG!!!"))

	startTime := time.Now()
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("WebSocket read error:", err)
			return err
		}
		fmt.Printf("queryLogs Received: %s\n", message)

		if time.Since(startTime) > time.Second*10 {
			break
		}
	}

	return nil
}

func queryLogsFile() error {
	pid := 0
	fmt.Print("Input queryLogs pid:")
	fmt.Scan(&pid)

	serverURL := fmt.Sprintf("%s/api/v1/log/system/process/%d/logfile", http_url, pid)

	client := &http.Client{}
	req, err := http.NewRequest("GET", serverURL, nil)
	if err != nil {
		fmt.Println("queryLogsFile NewRequest failed:", err)
		return err
	}
	req.Header.Add("Authorization", token)
	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("queryLogsFile failed:", err)
		return err
	}

	// dataInfo, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	fmt.Println("ReadAll:", err)
	// 	return err
	// }

	contentDisposition := resp.Header.Get("Content-Disposition")
	fileName := extractFileName(contentDisposition)
	fmt.Println("extractFileName:", fileName, ",resp header:", resp.Header)
	if fileName == "" {
		fileName = strconv.FormatInt(int64(pid), 10) + ".tgz"
	}
	localFile, err := os.Create("abc_" + fileName)
	if err != nil {
		fmt.Println("Create file ", fileName, " failed:", err)
		return err
	}
	defer localFile.Close()

	size, err := io.Copy(localFile, resp.Body)
	if err != nil {
		fmt.Println("Copy Error:", err)
		return err
	}

	fmt.Println("queryLogsFile size:", size)
	return nil
}

func getOverView() error {
	model := "ECO-E215WS"
	serverURL := fmt.Sprintf("%s/api/v1/log/overview/%s", http_url, model)

	client := &http.Client{}
	req, err := http.NewRequest("GET", serverURL, nil)
	if err != nil {
		fmt.Println("getOverView NewRequest failed:", err)
		return err
	}
	req.Header.Add("Authorization", token)
	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("getOverView failed:", err)
		return err
	}
	dataInfo, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ReadAll:", err)
		return err
	}

	fmt.Println("getOverView:", string(dataInfo))
	return nil
}

func main() {
	err := login()
	if err != nil {
		fmt.Println(">>>login failed:", err.Error())
		return
	}
	// WebSocket 服务器地址
	token = "Bearer " + token
	utlstr := url.QueryEscape(token)
	serverURL := ws_url + "/api/v1/log/ws/push?authorization=" + utlstr

	// 解析服务器地址

	u, err := url.Parse(serverURL)
	if err != nil {
		fmt.Println("Invalid server URL:", err)
		return
	}
	fmt.Println("server push url:", u.String())
	header := http.Header{}
	header.Add("Authorization", token)

	// 创建 WebSocket 连接
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Println("WebSocket connection failed:", err)
		return
	}
	defer conn.Close()

	num := 0

	// 创建一个信号通道，以便捕捉操作系统信号
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// 启动一个 goroutine 来接收 WebSocket 消息
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("WebSocket read error:", err)
				return
			}
			fmt.Printf("Received: %s\n", message)

			dataInfo := make(map[string]interface{})
			err = json.Unmarshal(message, &dataInfo)
			if err != nil {
				fmt.Println("Unmarshal body err:", err)
				continue
			}
			if dataInfo["topic"] == "logdump" {
				content := dataInfo["content"]
				contentMap, ok := content.(map[string]interface{})
				if !ok {
					fmt.Println("content not map:", string(message))
					continue
				}
				taskid := contentMap["taskid"]
				var taskID int64
				switch value := taskid.(type) {
				case int:
					taskID = int64(value)
				case int64:
					taskID = value
				case float64:
					taskID = int64(value)
				default:
					fmt.Println("taskid not int or int64:", string(message))
					return
				}

				queryGetLogDump(taskID)
			}
		}
	}()

	// 循环发送消息
	for {

		select {
		case <-interrupt:
			fmt.Println("Interrupt signal received. Closing WebSocket.")
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		case <-time.After(2 * time.Second):
			if num < 30 {
				// go queryProjSn()

				// //go getAllOnlines()
				// go queryDev()
				// go queryEmsYX()
				//queryMqttIdHis()

				// err, taskid := queryLogDumpStart()
				// if err != nil {
				// 	fmt.Println("queryLogDumpStart err:", err.Error())
				// 	break
				// }
				// time.Sleep(350 * time.Second)
				// err, _ = queryLogDumpStop(taskid)
				// if err != nil {
				// 	fmt.Println("queryLogDumpStop err:", err.Error())
				// 	break
				// }

				//queryProclistGet()
				//queryProclistPost()//////////////////////////////
				// queryModbusTransTab()
				// queryModbusTab()
				//queryLocalProcs()
				// queryLogTypes()
				// queryLogs()
				//queryLogsFile()
				// queryLocalProcs()
				// queryGW()
				// queryYX2()
				getOverView()
				break
			}

			if num > 3000000000 {
				fmt.Println("num 3 received. Closing WebSocket.")
				conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				break
			}
			message := []byte("Hello, WebSocket Server!")
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				fmt.Println("WebSocket write error:", err)
				return
			}
			fmt.Println("Sent: Hello, WebSocket Server!")
		}
		num++
	}
}
