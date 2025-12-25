# 用户数据验证与规范化功能

## 概述

本功能在从企业微信、钉钉、飞书同步员工数据到 go-ldap-admin/LDAP 之前，对用户数据进行严格的验证和规范化处理，解决以下问题：

1. **中文姓名拼音冲突**：如"王雨"和"王禹"拼音相同导致的 username/uid 冲突
2. **邮箱特殊字符**：如 `wang·yu@hzxb.com` 中的特殊字符导致的同步失败
3. **用户名唯一性保证**：通过追加手机号后四位确保唯一性

## 业务规则

### 1. 用户名唯一性策略

- username 与 uid 必须一致且在 LDAP 中唯一
- 用户名转换为小写，去除空格
- 若拼音 base 已存在，追加手机号后4位：`{base}{phone[-4:]}`
- 例如：`wangyu` 已存在，手机号 `18237009876` → `wangyu9876`

### 2. 邮箱验证与处理

- **清洗规则**：仅保留字母(A-Z/a-z)、数字(0-9)、点(.)、连字符(-)、下划线(_)
- **强制重写条件**：
  - 用户名被修改（因冲突追加了手机号）
  - 邮箱包含特殊字符（如中点·）
  - 邮箱格式不合法或为空
- **重写格式**：`{finalUsername}@{defaultDomain}`，默认域名为 `hzxb.com`

### 3. 执行时机

在写入 MySQL/LDAP 之前执行 `ValidateAndNormalizeUser()` 函数，确保：
- 先验证规范化
- 再写入 MySQL
- 最后同步到 LDAP

## 技术实现

### 核心函数

位置：`public/tools/user_validator.go`

1. **GetPhoneLast4Digits(phone string) string**
   - 获取手机号后4位
   - 不足4位返回空字符串

2. **SanitizeEmailLocalPart(localPart string) string**
   - 清洗邮箱本地部分（@之前）
   - 使用正则：`[^A-Za-z0-9.\-_]`

3. **ValidateEmail(email string) bool**
   - 验证邮箱格式
   - 正则：`^[A-Za-z0-9._%-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`

4. **NormalizeEmail(email, username, defaultDomain string) string**
   - 规范化邮箱地址
   - 清洗 local-part
   - 验证格式
   - 不合法则使用 `{username}@{defaultDomain}`

5. **GenerateUniqueUsername(baseUsername, phone string, checkExists func(string) bool) (string, bool)**
   - 生成唯一用户名
   - 检查冲突
   - 冲突时追加手机号后4位
   - 返回最终用户名和是否修改标志

6. **ValidateAndNormalizeUser(user *model.User, defaultEmailDomain string, checkExists func(string) bool) error**
   - 主函数，整合所有验证逻辑
   - 先规范化用户名
   - 再规范化邮箱
   - 更新 `user.Username` 和 `user.Mail`

### 集成点

以下三个同步逻辑文件已集成验证：

1. `logic/wecom_logic.go` - WeComLogic.AddUsers()
2. `logic/dingtalk_logic.go` - DingTalkLogic.AddUsers()
3. `logic/feishu_logic.go` - FeiShuLogic.AddUsers()

**集成代码示例**：

```go
// 在写入MySQL/LDAP之前，进行用户数据验证和规范化
defaultEmailDomain := "hzxb.com"
if config.Conf.Ldap.DefaultEmailSuffix != "" {
    defaultEmailDomain = config.Conf.Ldap.DefaultEmailSuffix
}

// 用户名存在性检查函数
checkUsernameExists := func(username string) bool {
    return isql.User.Exist(tools.H{"username": username})
}

// 验证并规范化用户数据（用户名唯一性 + 邮箱清洗）
err := tools.ValidateAndNormalizeUser(user, defaultEmailDomain, checkUsernameExists)
if err != nil {
    return tools.NewValidatorError(fmt.Errorf("用户数据验证失败:%s", err.Error()))
}

// 记录数据规范化日志
common.Log.Infof("用户数据规范化完成: username=%s, mail=%s, mobile=%s", user.Username, user.Mail, user.Mobile)
```

## 测试覆盖

### 单元测试

位置：`public/tools/user_validator_test.go`

- TestGetPhoneLast4Digits：手机号后4位提取
- TestSanitizeEmailLocalPart：邮箱特殊字符清洗
- TestValidateEmail：邮箱格式验证
- TestNormalizeEmail：邮箱规范化
- TestGenerateUniqueUsernameFormat：用户名格式转换
- TestGenerateUniqueUsernameWithConflict：用户名冲突处理

### 集成测试

位置：`public/tools/user_validator_integration_test.go`

测试场景包括：
- 场景1：用户名冲突，邮箱正常
- 场景2：用户名冲突，邮箱含特殊字符
- 场景3：新用户，无冲突
- 场景4：邮箱完全无效
- 场景5：空邮箱
- 场景6：用户名大小写混合

所有测试用例均通过。

## 实际案例

### 案例1：王禹

**输入**：
- 姓名：王禹
- 拼音：wangyu
- 手机号：18237009876
- 邮箱：wangyu@hzxb.com

**输出**（假设 wangyu 已存在）：
- username/uid：wangyu9876
- mail：wangyu9876@hzxb.com

### 案例2：王雨

**输入**：
- 姓名：王雨
- 拼音：wangyu
- 手机号：18237001122
- 邮箱：wang·yu@hzxb.com

**输出**（假设 wangyu 已存在）：
- username/uid：wangyu1122
- mail：wangyu1122@hzxb.com

## 配置说明

默认邮箱域名配置：

```yaml
ldap:
  default-email-suffix: "hzxb.com"  # 在 config.yml 中配置
```

或通过环境变量：

```bash
export LDAP_DEFAULT_EMAIL_SUFFIX="hzxb.com"
```

## 安全性

- 通过 CodeQL 安全扫描：0 个安全漏洞
- 输入验证：所有用户输入经过清洗和验证
- 正则表达式：经过优化，防止 ReDoS 攻击
- 数据完整性：确保 username 和 email 符合规范

## 并发安全

- 用户名唯一性检查通过数据库唯一索引保证
- `username` 字段在数据库中已设置 `unique` 约束
- 建议：在高并发场景下，数据库插入失败时可实现重试机制

## 注意事项

1. **手机号要求**：必须至少4位才能生成唯一后缀
2. **已存在用户**：功能仅在新增用户时生效，已存在用户的更新逻辑未修改
3. **日志记录**：所有规范化操作都有日志记录，便于审计
4. **向后兼容**：不影响现有用户数据

## 未来改进

1. 支持自定义唯一性后缀策略（当前仅支持手机号后4位）
2. 提供管理界面显示规范化历史
3. 支持批量修正已存在的不规范数据
4. 增加邮箱验证（发送验证邮件）
