package utils

import (
	"regexp"
	"time"
	"unicode/utf8"
)

// 全局预编译所有正则，程序启动只执行一次
var (
	// 昵称正则
	regUserValid = regexp.MustCompile(`^[一-龥a-zA-Z0-9_-]+$`)

	// 密码正则
	regPwdChinese = regexp.MustCompile(`[一-龥]`)
	regPwdLetter  = regexp.MustCompile(`[A-Za-z]`)
	regPwdDigit   = regexp.MustCompile(`\d`)
	regPwdSpace   = regexp.MustCompile(` `)

	// QQ邮箱正则
	regQQEmail = regexp.MustCompile(`^[1-9]\d{4,10}@qq\.com$`)

	// 6位数字验证码
	regCode6 = regexp.MustCompile(`^\d{6}$`)

	regIntro = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// CheckNickname 昵称校验
func CheckNickname(value string) string {
	if value == "" {
		return "昵称不能为空"
	}

	if len(value) < 1 || len(value) > 20 {
		return "昵称长度1-20位"
	}

	if !regUserValid.MatchString(value) {
		return "昵称仅支持中文、字母、数字、_、-"
	}
	return ""
}

// CheckPassword 密码校验
func CheckPassword(value string) string {
	if value == "" {
		return "请输入密码"
	}
	if regPwdChinese.MatchString(value) {
		return "密码不能包含中文"
	}
	if len(value) < 8 || len(value) > 20 {
		return "密码长度 8-20 位"
	}
	if !regPwdLetter.MatchString(value) {
		return "必须包含字母"
	}
	if !regPwdDigit.MatchString(value) {
		return "必须包含数字"
	}
	if regPwdSpace.MatchString(value) {
		return "密码不能包含空格"
	}
	return ""
}

// CheckEmail QQ邮箱校验
func CheckEmail(value string) string {
	if value == "" {
		return "邮箱不能为空"
	}
	if !regQQEmail.MatchString(value) {
		return "请输入正确的QQ邮箱格式"
	}
	return ""
}

// CheckCode 6位数字验证码
func CheckCode(value string) string {
	if value == "" {
		return "请输入验证码"
	}
	if !regCode6.MatchString(value) {
		return "必须是6位数字"
	}
	return ""
}

func CheckIntro(value string) string {
	// 不能直接用 len(value) 因为 len(value) 返回的是字节数量 英文字母、数字、半角符号：1 字节 / 个
	// 中文汉字、中文全角逗号、句号：UTF-8 编码占 3 字节 / 个
	count := utf8.RuneCountInString(value)
	if count > 100 {
		return "简介不能超过100个字"
	}

	if regIntro.MatchString(value) {
		return "简介包含非法字符 < >"
	}
	return ""

}

func CheckGender(value int8) string {
	switch value {
	case 0, 1, 2:
		return ""
	default:
		return "性别参数不合法"
	}
}

func CheckBirthday(dateStr string) string {
	if dateStr == "" {
		return ""
	}

	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return "生日格式必须为 yyyy-MM-dd"
	}

	now := time.Now()
	minBirth := now.AddDate(-120, 0, 0)
	maxBirth := now
	if t.Before(minBirth) || t.After(maxBirth) {
		return "生日日期超出合法范围"
	}
	return ""
}

func CheckVerifyMsg(value string) string {
	count := utf8.RuneCountInString(value)
	if count == 0 {
		return "请输入验证信息"
	}
	if count > 80 {
		return "验证信息长度不能超过 80"
	}
	return ""
}
