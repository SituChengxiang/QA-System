package model

// AdminType 定义管理员类型
type AdminType int

const (
	// NormalAdmin 普通管理员
	NormalAdmin AdminType = 1
	// SuperAdmin 超级管理员
	SuperAdmin AdminType = 2
)

// User 用户模型
type User struct {
	ID          int    `json:"id"`           // 用户id
	Username    string `json:"username"`     // 用户名
	Password    string `json:"password"`     // 密码
	AdminType   int    `json:"admin_type"`   // 管理员类型
	NotifyEmail string `json:"notify_email"` // 提醒邮箱
}
