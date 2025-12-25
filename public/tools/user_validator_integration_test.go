package tools

import (
	"testing"

	"github.com/eryajf/go-ldap-admin/model"
)

func TestValidateAndNormalizeUser_IntegrationScenarios(t *testing.T) {
	// 模拟问题描述中的场景
	tests := []struct {
		name              string
		inputUser         *model.User
		defaultDomain     string
		existingUsernames map[string]bool // 模拟已存在的用户名
		expectedUsername  string
		expectedMail      string
		description       string
	}{
		{
			name: "场景1: 王禹 - 用户名冲突，邮箱正常",
			inputUser: &model.User{
				Username: "wangyu",
				Mobile:   "18237009876",
				Mail:     "wangyu@hzxb.com",
			},
			defaultDomain: "hzxb.com",
			existingUsernames: map[string]bool{
				"wangyu": true, // wangyu已存在
			},
			expectedUsername: "wangyu9876",
			expectedMail:     "wangyu9876@hzxb.com", // 使用规范化后的username
			description:      "王禹的拼音wangyu已被占用，追加手机号后4位",
		},
		{
			name: "场景2: 王雨 - 用户名冲突，邮箱含特殊字符",
			inputUser: &model.User{
				Username: "wangyu",
				Mobile:   "18237001122",
				Mail:     "wang·yu@hzxb.com",
			},
			defaultDomain: "hzxb.com",
			existingUsernames: map[string]bool{
				"wangyu": true, // wangyu已存在
			},
			expectedUsername: "wangyu1122",
			expectedMail:     "wangyu1122@hzxb.com", // 清洗后的邮箱，使用规范化后的username
			description:      "王雨的拼音wangyu已被占用，追加手机号后4位；邮箱含特殊字符·，需清洗",
		},
		{
			name: "场景3: 新用户 - 用户名不冲突，邮箱正常",
			inputUser: &model.User{
				Username: "zhangsan",
				Mobile:   "18237003333",
				Mail:     "zhangsan@hzxb.com",
			},
			defaultDomain:     "hzxb.com",
			existingUsernames: map[string]bool{},
			expectedUsername:  "zhangsan",
			expectedMail:      "zhangsan@hzxb.com",
			description:       "新用户，用户名不冲突，邮箱格式正常，不需要修改",
		},
		{
			name: "场景4: 邮箱完全无效，使用username@domain",
			inputUser: &model.User{
				Username: "lisi",
				Mobile:   "18237004444",
				Mail:     "····@invalid",
			},
			defaultDomain:     "hzxb.com",
			existingUsernames: map[string]bool{},
			expectedUsername:  "lisi",
			expectedMail:      "lisi@hzxb.com",
			description:       "邮箱local-part全是特殊字符，使用username@domain",
		},
		{
			name: "场景5: 空邮箱，使用默认格式",
			inputUser: &model.User{
				Username: "wangwu",
				Mobile:   "18237005555",
				Mail:     "",
			},
			defaultDomain:     "hzxb.com",
			existingUsernames: map[string]bool{},
			expectedUsername:  "wangwu",
			expectedMail:      "wangwu@hzxb.com",
			description:       "邮箱为空，使用默认格式 username@domain",
		},
		{
			name: "场景6: 用户名大小写混合，需要规范化",
			inputUser: &model.User{
				Username: "WangYu",
				Mobile:   "18237006666",
				Mail:     "WangYu@hzxb.com",
			},
			defaultDomain: "hzxb.com",
			existingUsernames: map[string]bool{
				"wangyu": true, // wangyu已存在
			},
			expectedUsername: "wangyu6666",
			expectedMail:     "wangyu6666@hzxb.com",
			description:      "用户名大小写转小写后冲突，追加手机号后4位",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建检查函数
			checkExists := func(username string) bool {
				return tt.existingUsernames[username]
			}

			// 执行验证和规范化
			err := ValidateAndNormalizeUser(tt.inputUser, tt.defaultDomain, checkExists)
			if err != nil {
				t.Fatalf("ValidateAndNormalizeUser() 返回错误: %v", err)
			}

			// 验证用户名
			if tt.inputUser.Username != tt.expectedUsername {
				t.Errorf("Username = %s; want %s (描述: %s)",
					tt.inputUser.Username, tt.expectedUsername, tt.description)
			}

			// 验证邮箱
			if tt.inputUser.Mail != tt.expectedMail {
				t.Errorf("Mail = %s; want %s (描述: %s)",
					tt.inputUser.Mail, tt.expectedMail, tt.description)
			}
		})
	}
}

func TestValidateAndNormalizeUser_EdgeCases(t *testing.T) {
	tests := []struct {
		name              string
		inputUser         *model.User
		defaultDomain     string
		existingUsernames map[string]bool
		shouldModify      bool
		description       string
	}{
		{
			name: "手机号不足4位，无法生成唯一用户名",
			inputUser: &model.User{
				Username: "test",
				Mobile:   "123", // 不足4位
				Mail:     "test@hzxb.com",
			},
			defaultDomain: "hzxb.com",
			existingUsernames: map[string]bool{
				"test": true, // test已存在
			},
			shouldModify: false, // 由于手机号不足4位，无法修改用户名
			description:  "手机号不足4位时，即使用户名冲突也无法生成唯一用户名",
		},
		{
			name: "带空格的用户名需要去除",
			inputUser: &model.User{
				Username: "wang yu",
				Mobile:   "18237007777",
				Mail:     "wangyu@hzxb.com",
			},
			defaultDomain:     "hzxb.com",
			existingUsernames: map[string]bool{},
			shouldModify:      true,
			description:       "用户名中的空格应该被去除",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkExists := func(username string) bool {
				return tt.existingUsernames[username]
			}

			err := ValidateAndNormalizeUser(tt.inputUser, tt.defaultDomain, checkExists)
			if err != nil {
				t.Fatalf("ValidateAndNormalizeUser() 返回错误: %v", err)
			}

			t.Logf("测试 '%s': username=%s, mail=%s", tt.name, tt.inputUser.Username, tt.inputUser.Mail)
		})
	}
}
