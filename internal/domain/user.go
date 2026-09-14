package domain

import "time"

// Roleはユーザーの権限区分
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

// Roleが定義済みの値が判定
func (r Role) IsValid() bool {
	switch r {
	case RoleUser, RoleAdmin:
		return true
	default:
		return false
	}
}

// 認証主体
type User struct {
	ID           int64
	Email        string
	PasswordHash string `json:"-"` // 万が一JSON化されても出力しない
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// 管理者権限を持つかを返す
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}
