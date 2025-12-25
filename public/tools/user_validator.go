package tools

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/eryajf/go-ldap-admin/model"
)

var (
	// 邮箱验证正则：标准邮箱格式
	emailRegex = regexp.MustCompile(`^[A-Za-z0-9._%-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`)
	// 邮箱本地部分特殊字符清洗正则：仅保留字母、数字、点、连字符、下划线
	emailLocalPartCleanRegex = regexp.MustCompile(`[^A-Za-z0-9.\-_]`)
)

// GetPhoneLast4Digits 获取手机号后4位
// 如果手机号不足4位，返回空字符串
func GetPhoneLast4Digits(phone string) string {
	phone = strings.TrimSpace(phone)
	if len(phone) < 4 {
		return ""
	}
	return phone[len(phone)-4:]
}

// SanitizeEmailLocalPart 清洗邮箱的本地部分（@之前的部分）
// 仅保留字母（A-Z/a-z）、数字（0-9）、点（.）、连字符（-）和下划线（_）
func SanitizeEmailLocalPart(localPart string) string {
	return emailLocalPartCleanRegex.ReplaceAllString(localPart, "")
}

// ValidateEmail 验证邮箱格式是否合法
func ValidateEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// NormalizeEmail 规范化邮箱地址
// 1. 如果邮箱为空，使用 username@domain 格式
// 2. 如果邮箱不为空，清洗 local-part 并验证
// 3. 如果清洗后仍不合法，强制使用 username@domain
func NormalizeEmail(email, username, defaultDomain string) string {
	// 如果邮箱为空，直接使用默认格式
	if email == "" {
		return fmt.Sprintf("%s@%s", username, defaultDomain)
	}

	// 分离邮箱的 local-part 和 domain
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		// 格式不正确，使用默认格式
		return fmt.Sprintf("%s@%s", username, defaultDomain)
	}

	localPart := parts[0]
	domain := parts[1]

	// 清洗 local-part
	cleanedLocalPart := SanitizeEmailLocalPart(localPart)

	// 如果清洗后为空，使用 username
	if cleanedLocalPart == "" {
		cleanedLocalPart = username
		domain = defaultDomain
	}

	// 重新组合邮箱
	normalizedEmail := fmt.Sprintf("%s@%s", cleanedLocalPart, domain)

	// 验证清洗后的邮箱是否合法
	if !ValidateEmail(normalizedEmail) {
		// 如果仍不合法，强制使用默认格式
		return fmt.Sprintf("%s@%s", username, defaultDomain)
	}

	return normalizedEmail
}

// GenerateUniqueUsername 生成唯一的用户名
// 如果 baseUsername 已存在，追加手机号后4位
// checkExists: 检查用户名是否存在的函数
// 返回：最终用户名和是否进行了修改
func GenerateUniqueUsername(baseUsername, phone string, checkExists func(string) bool) (string, bool) {
	// 转换为小写并去除空格
	baseUsername = strings.ToLower(strings.ReplaceAll(baseUsername, " ", ""))

	// 检查基础用户名是否存在
	if !checkExists(baseUsername) {
		return baseUsername, false
	}

	// 如果存在，追加手机号后4位
	last4 := GetPhoneLast4Digits(phone)
	if last4 == "" {
		// 如果无法获取手机号后4位，无法生成唯一用户名
		// 这种情况下返回原用户名，让后续流程处理冲突
		return baseUsername, false
	}

	finalUsername := baseUsername + last4
	return finalUsername, true
}

// ValidateAndNormalizeUser 验证并规范化用户数据
// 在写入 MySQL/LDAP 之前调用
// checkExists: 检查用户名是否存在的函数
// defaultEmailDomain: 默认邮箱域名（如 "hzxb.com"）
// 1. 规范化用户名（确保唯一性）
// 2. 规范化邮箱（清洗特殊字符、验证格式）
// 3. 更新 user.Username、user.Mail
func ValidateAndNormalizeUser(user *model.User, defaultEmailDomain string, checkExists func(string) bool) error {
	// 1. 生成唯一用户名
	finalUsername, modified := GenerateUniqueUsername(user.Username, user.Mobile, checkExists)
	user.Username = finalUsername

	// 2. 规范化邮箱
	originalEmail := user.Mail
	
	// 如果用户名被修改了，或者邮箱包含特殊字符，需要强制使用新的username
	needsEmailUpdate := modified
	
	// 检查邮箱是否包含需要清洗的字符
	if originalEmail != "" {
		parts := strings.Split(originalEmail, "@")
		if len(parts) == 2 {
			cleanedLocalPart := SanitizeEmailLocalPart(parts[0])
			if cleanedLocalPart != parts[0] {
				// 邮箱包含需要清洗的特殊字符
				needsEmailUpdate = true
			}
		}
	}
	
	// 如果需要更新邮箱，强制使用 finalUsername@defaultDomain
	if needsEmailUpdate || originalEmail == "" {
		user.Mail = fmt.Sprintf("%s@%s", finalUsername, defaultEmailDomain)
	} else {
		// 否则，进行常规的邮箱规范化
		normalizedEmail := NormalizeEmail(originalEmail, finalUsername, defaultEmailDomain)
		user.Mail = normalizedEmail
	}

	return nil
}
