// Package hash 封装密码哈希（bcrypt）。
package hash

import "golang.org/x/crypto/bcrypt"

// Hash 对明文密码生成 bcrypt 哈希。
func Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Compare 校验明文与哈希。匹配返回 nil。
func Compare(hashed, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password))
}
