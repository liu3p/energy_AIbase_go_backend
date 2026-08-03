package models

//nolint:govet
type UserInfo struct {
	UserName  string `json:"username"`
	UserType  string `json:"usertype"`
	Status    int    `json:"status"`
	LastLogin string `json:"lastlogin"`
}
