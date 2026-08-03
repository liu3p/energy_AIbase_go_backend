package emsclient

/*
import (
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"sort"
	"webserver-arm/log"
	"webserver-arm/models"

	"github.com/goccy/go-json"

	"golang.org/x/crypto/ssh"

	"github.com/pkg/sftp"
)

const (
	// redis call  700-799
	RedisCallMethod        = 701
	RedisCallHistoryMethod = 702
	RedisCallLogDumpMethod = 703
	RedisCallProcMethod    = 704
)

//nolint:govet
type RedisCallReq struct {
	Seq     int64  `json:"seq"`
	Method  string `json:"method"`
	Payload string `json:"payload"`
}

func (e *RedisCallReq) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
func (e *RedisCallReq) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *EmsClient) RedisCall(req *RedisCallReq) (resp *models.CommonTransResp, err error) {
	method := uint32(RedisCallMethod)
	resp = &models.CommonTransResp{}
	err = e.Call(method, req, resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

//nolint:govet
type RedisCallHisReq struct {
	Seq     int64  `json:"seq"`
	DevSn   string `json:"devsn"`
	Mqttid  string `json:"mqttid"`
	Payload string `json:"payload"`
}

func (e *RedisCallHisReq) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
func (e *RedisCallHisReq) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *EmsClient) RedisCallHistory(req *RedisCallHisReq) (resp *models.CommonTransResp, err error) {
	method := uint32(RedisCallHistoryMethod)
	resp = &models.CommonTransResp{}
	err = e.Call(method, req, resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

//nolint:govet
type RedisCallLogDumpReq struct {
	Seq      int64  `json:"seq"`
	TaskID   int64  `json:"taskid"`
	Duration int    `json:"duration"`
	HostIP   string `json:"hostip"`
	Stop     int    `json:"stop"`
}

func (e *RedisCallLogDumpReq) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
func (e *RedisCallLogDumpReq) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *EmsClient) RedisCallLogDump(req *RedisCallLogDumpReq) (resp *models.CommonTransResp, err error) {
	method := uint32(RedisCallLogDumpMethod)
	resp = &models.CommonTransResp{}
	err = e.Call(method, req, resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

//nolint:govet
type PushLogDumpTask struct {
	TaskID  int64  `json:"taskid"`
	Result  int    `json:"result"`
	HostIP  string `json:"hostip"`
	LogPath string `json:"logpath"`
}

func (p *PushLogDumpTask) clean() {
	p.TaskID = 0
	p.Result = 0
	p.HostIP = ""
	p.LogPath = ""
}

func (p *PushLogDumpTask) Marshal() ([]byte, error) {
	return json.Marshal(p)
}
func (p *PushLogDumpTask) Unmarshal(data []byte) error {
	p.clean()
	log.Debug(">>>>>>> PushLogDumpTask:", string(data))
	return json.Unmarshal(data, p)
}
func (p *PushLogDumpTask) Clone() models.IEmsRPCResp {
	return &PushLogDumpTask{p.TaskID, p.Result, p.HostIP, p.LogPath}
}

type PushLogDumpResp struct {
	TaskID int64 `json:"taskid"`
	Result int   `json:"result"`
}

type SFTPFileFetcher struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

func NewSFTPFileFetcher(hostIP, user, pwd string) (*SFTPFileFetcher, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pwd),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", hostIP), config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial SSH server: %w", err)
	}

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create SCP client: %w", err)
	}

	return &SFTPFileFetcher{sshClient: client, sftpClient: sftpClient}, nil
}

func (s *SFTPFileFetcher) Close() {
	s.sftpClient.Close()
	s.sshClient.Close()
}

func (s *SFTPFileFetcher) CopyRemoteToLocal(remotePath, localPath string) error {
	fileInfo, err := s.sftpClient.Stat(remotePath)
	if err != nil {
		return fmt.Errorf("failed to stat remote path %s info: %w", remotePath, err)
	}

	if fileInfo.IsDir() {
		err = s.copyRemoteDirectory(remotePath, localPath)
	} else {
		err = s.copyRemoteFile(remotePath, localPath)
	}

	return err
}

func (s *SFTPFileFetcher) copyRemoteDirectory(remotePath, localPath string) error {
	// 创建本地目录
	err := os.MkdirAll(localPath, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create local directory %s: %w", localPath, err)
	}
	// 读取远程目录中的文件
	remoteFiles, err := s.sftpClient.ReadDir(remotePath)
	if err != nil {
		return fmt.Errorf("failed to read remote directory %s: %w", remotePath, err)
	}

	// 递归拷贝每个文件或子目录
	for _, remoteFile := range remoteFiles {
		remoteFilePath := filepath.Join(remotePath, remoteFile.Name())
		localFilePath := filepath.Join(localPath, remoteFile.Name())

		if remoteFile.IsDir() {
			// 递归拷贝子目录
			err = s.copyRemoteDirectory(remoteFilePath, localFilePath)
			if err != nil {
				return err
			}
		} else {
			// 拷贝文件
			err = s.copyRemoteFile(remoteFilePath, localFilePath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *SFTPFileFetcher) copyRemoteFile(remoteFilePath, localFilePath string) error {
	remoteFile, err := s.sftpClient.Open(remoteFilePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file %s: %w", remoteFilePath, err)
	}
	defer remoteFile.Close()
	localFile, err := os.Create(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to create local file %s: %w", localFilePath, err)
	}
	defer localFile.Close()

	_, err = io.Copy(localFile, remoteFile)
	if err != nil {
		return fmt.Errorf("failed to copy remote file %s to local file %s: %w", remoteFilePath, localFilePath, err)
	}
	return nil
}

func (s *SFTPFileFetcher) CopyLocalFileToDir(srcPath, dstDirPath string) error {
	fileName := filepath.Base(srcPath)
	dstPath := filepath.Join(dstDirPath, fileName)
	return s.CopyLocalFile(srcPath, dstPath)
}

func (s *SFTPFileFetcher) CopyLocalFile(srcPath, dstPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", srcPath, err)
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create target file %s: %w", dstPath, err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy source file %s to target file %s: %w", srcPath, dstPath, err)
	}
	return nil
}

func deleteOldFiles(directoryPath string, maxFilesToKeep int) error {
	log.Info("start deleteOldFiles for dir:", directoryPath)
	dirEntrys, err := os.ReadDir(directoryPath)
	if err != nil {
		log.Error("deleteOldFiles failed. ReadDir err:", err.Error())
		return err
	}

	fileInfoList := make([]fs.FileInfo, 0, len(dirEntrys))

	for _, entry := range dirEntrys {
		fileInfo, err := entry.Info()
		if err != nil {
			log.Error("get entry info for ", entry.Name(), " failed:", err.Error())
			continue
		}
		fileInfoList = append(fileInfoList, fileInfo)
	}

	// 按修改时间降序排序
	sort.Slice(fileInfoList, func(i, j int) bool {
		return fileInfoList[i].ModTime().After(fileInfoList[j].ModTime())
	})

	// 保留最新的maxFilesToKeep个文件，删除其余文件
	for i := maxFilesToKeep; i < len(fileInfoList); i++ {
		filePath := filepath.Join(directoryPath, fileInfoList[i].Name())
		log.Error("Deleting old file:", filePath)
		err := os.RemoveAll(filePath)
		if err != nil {
			log.Errorf("Error deleting file %s: %w", filePath, err)
		}
	}

	return nil
}

//nolint:govet
type RedisCallProcReq struct {
	Seq    int64  `json:"seq"`
	Method string `json:"method"`
	Body   string `json:"body"`
}

func (e *RedisCallProcReq) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
func (e *RedisCallProcReq) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *EmsClient) RedisCallProc(req *RedisCallProcReq) (resp *models.CommonTransResp, err error) {
	method := uint32(RedisCallProcMethod)
	resp = &models.CommonTransResp{}
	err = e.Call(method, req, resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}
*/
