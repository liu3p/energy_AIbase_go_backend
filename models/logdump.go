package models

import (
	"os"
	"path"
	"strconv"
)

const (
	LogDumpSubPath = "/dat/logdump_lw"
	LogDumpUser    = "prs3000"
	LogDumpPwd     = "prs3000"
	LogFileSubPath = "/dat/log"
)

var LogDumpPath = path.Join(os.Getenv("ECOEMS"), LogDumpSubPath)
var SysLogFilePath = path.Join(os.Getenv("ECOEMS"), LogFileSubPath)

func LogDumpLocalPath(taskid int64) string {
	return path.Join(LogDumpPath, strconv.FormatInt(taskid, 10))
}

func LogDumpLocalPkgFile(taskid int64) string {
	fileName := "logdump_" + strconv.FormatInt(taskid, 10) + ".tgz"
	return path.Join(LogDumpPath, fileName)
}

func LogDumpLocalPkgFileE(taskid string) string {
	fileName := "logdump_" + taskid + ".tgz"
	return path.Join(LogDumpPath, fileName)
}
