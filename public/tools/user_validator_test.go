package tools

import (
	"testing"
)

func TestGetPhoneLast4Digits(t *testing.T) {
	tests := []struct {
		name     string
		phone    string
		expected string
	}{
		{"正常手机号", "18237009876", "9876"},
		{"正常手机号2", "18237001122", "1122"},
		{"少于4位", "123", ""},
		{"刚好4位", "1234", "1234"},
		{"带空格", "  18237009876  ", "9876"},
		{"空字符串", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPhoneLast4Digits(tt.phone)
			if result != tt.expected {
				t.Errorf("GetPhoneLast4Digits(%s) = %s; want %s", tt.phone, result, tt.expected)
			}
		})
	}
}

func TestSanitizeEmailLocalPart(t *testing.T) {
	tests := []struct {
		name      string
		localPart string
		expected  string
	}{
		{"正常邮箱", "wangyu", "wangyu"},
		{"带点的邮箱", "wang.yu", "wang.yu"},
		{"带中点的邮箱", "wang·yu", "wangyu"},
		{"带多个特殊字符", "wang·yu@test", "wangyutest"},
		{"仅特殊字符", "····", ""},
		{"中英文混合", "王雨wangyu", "wangyu"},
		{"带下划线和连字符", "wang_yu-test", "wang_yu-test"},
		{"带数字", "wang123", "wang123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeEmailLocalPart(tt.localPart)
			if result != tt.expected {
				t.Errorf("SanitizeEmailLocalPart(%s) = %s; want %s", tt.localPart, result, tt.expected)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		{"正常邮箱", "wangyu@hzxb.com", true},
		{"带点的邮箱", "wang.yu@hzxb.com", true},
		{"带数字", "wang123@hzxb.com", true},
		{"带下划线", "wang_yu@hzxb.com", true},
		{"带连字符", "wang-yu@hzxb.com", true},
		{"无@符号", "wangyuhzxb.com", false},
		{"无域名", "wangyu@", false},
		{"无本地部分", "@hzxb.com", false},
		{"空字符串", "", false},
		{"域名无后缀", "wangyu@hzxb", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateEmail(tt.email)
			if result != tt.expected {
				t.Errorf("ValidateEmail(%s) = %v; want %v", tt.email, result, tt.expected)
			}
		})
	}
}

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		username      string
		defaultDomain string
		expected      string
	}{
		{
			name:          "正常邮箱",
			email:         "wangyu@hzxb.com",
			username:      "wangyu",
			defaultDomain: "hzxb.com",
			expected:      "wangyu@hzxb.com",
		},
		{
			name:          "带特殊字符的邮箱-需要清洗",
			email:         "wang·yu@hzxb.com",
			username:      "wangyu",
			defaultDomain: "hzxb.com",
			expected:      "wangyu@hzxb.com",
		},
		{
			name:          "空邮箱-使用默认",
			email:         "",
			username:      "wangyu",
			defaultDomain: "hzxb.com",
			expected:      "wangyu@hzxb.com",
		},
		{
			name:          "无效格式-使用默认",
			email:         "invalidemail",
			username:      "wangyu",
			defaultDomain: "hzxb.com",
			expected:      "wangyu@hzxb.com",
		},
		{
			name:          "local-part全是特殊字符-使用username",
			email:         "····@test.com",
			username:      "wangyu",
			defaultDomain: "hzxb.com",
			expected:      "wangyu@hzxb.com",
		},
		{
			name:          "带数字和下划线",
			email:         "wang_123@hzxb.com",
			username:      "wangyu",
			defaultDomain: "hzxb.com",
			expected:      "wang_123@hzxb.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeEmail(tt.email, tt.username, tt.defaultDomain)
			if result != tt.expected {
				t.Errorf("NormalizeEmail(%s, %s, %s) = %s; want %s",
					tt.email, tt.username, tt.defaultDomain, result, tt.expected)
			}
		})
	}
}

func TestGenerateUniqueUsernameFormat(t *testing.T) {
	// Mock function that always returns false (username doesn't exist)
	mockCheckExists := func(username string) bool {
		return false
	}

	tests := []struct {
		name         string
		baseUsername string
		phone        string
		expected     string
	}{
		{
			name:         "正常拼音-小写",
			baseUsername: "WangYu",
			phone:        "18237009876",
			expected:     "wangyu",
		},
		{
			name:         "带空格-去除空格",
			baseUsername: "wang yu",
			phone:        "18237009876",
			expected:     "wangyu",
		},
		{
			name:         "大写转小写",
			baseUsername: "WANGYU",
			phone:        "18237009876",
			expected:     "wangyu",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := GenerateUniqueUsername(tt.baseUsername, tt.phone, mockCheckExists)
			if result != tt.expected {
				t.Errorf("GenerateUniqueUsername(%s, %s) = %s; want %s", tt.baseUsername, tt.phone, result, tt.expected)
			}
		})
	}
}

func TestGenerateUniqueUsernameWithConflict(t *testing.T) {
	// Mock function that returns true for "wangyu" (exists)
	mockCheckExists := func(username string) bool {
		return username == "wangyu"
	}

	tests := []struct {
		name         string
		baseUsername string
		phone        string
		expected     string
		modified     bool
	}{
		{
			name:         "用户名冲突-追加手机号",
			baseUsername: "wangyu",
			phone:        "18237009876",
			expected:     "wangyu9876",
			modified:     true,
		},
		{
			name:         "用户名不冲突",
			baseUsername: "wangyu2",
			phone:        "18237009876",
			expected:     "wangyu2",
			modified:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, modified := GenerateUniqueUsername(tt.baseUsername, tt.phone, mockCheckExists)
			if result != tt.expected {
				t.Errorf("GenerateUniqueUsername(%s, %s) = %s; want %s", tt.baseUsername, tt.phone, result, tt.expected)
			}
			if modified != tt.modified {
				t.Errorf("GenerateUniqueUsername(%s, %s) modified = %v; want %v", tt.baseUsername, tt.phone, modified, tt.modified)
			}
		})
	}
}
