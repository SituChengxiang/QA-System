package utils

import (
	"errors"
	"time"

	global "QA-System/internal/global/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zjutjh/WeJH-SDK/oauth"
)

var key string

// UserClaims 用户信息
type UserClaims struct {
	Name         string `json:"name"`
	College      string `json:"college"`
	StudentID    string `json:"stuId"`
	UserType     string `json:"userType"`
	UserTypeDesc string `json:"userTypeDesc"`
	Gender       string `json:"gender"`
	jwt.RegisteredClaims
}

// NewJWT 生成 JWT
func NewJWT(name, college, stuId, userType, userTypeDesc, gender string) string {
	key = global.Config.GetString("jwt.key")
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, UserClaims{
		Name:         name,
		College:      college,
		StudentID:    stuId,
		UserType:     userType,
		UserTypeDesc: userTypeDesc,
		Gender:       gender,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(24*7) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	})
	s, err := t.SignedString([]byte(key))
	if err != nil {
		return ""
	}
	return s
}

// ParseJWT 解析 JWT
func ParseJWT(token string) (oauth.UserInfo, error) {
	key = global.Config.GetString("jwt.key")
	t, err := jwt.ParseWithClaims(token, &UserClaims{}, func(_ *jwt.Token) (any, error) {
		return []byte(key), nil
	})
	if err != nil {
		return oauth.UserInfo{}, err
	}

	userClaims, ok := t.Claims.(*UserClaims)
	if !ok || !t.Valid {
		return oauth.UserInfo{}, errors.New("invalid token")
	}

	userInfo := oauth.UserInfo{
		Name:         userClaims.Name,
		College:      userClaims.College,
		UserType:     userClaims.UserType,
		UserTypeDesc: userClaims.UserTypeDesc,
		Gender:       userClaims.Gender,
		StudentID:    userClaims.StudentID,
	}
	return userInfo, nil
}
