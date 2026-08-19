package utils

import (
	"ChatApp/config"
	"fmt"

	"gopkg.in/gomail.v2"
)

func SendQQEmail(toEmail string, code string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", config.Conf.Smtp.Account)
	m.SetHeader("To", toEmail)
	m.SetHeader("subject", fmt.Sprintf("验证码：%s", code))
	m.SetBody("text/html", `<p>您的验证码是：<b>`+code+`</b></p><p>有效期5分钟，请尽快使用。</p>`)
	d := gomail.NewDialer(
		config.Conf.Smtp.Host,
		config.Conf.Smtp.Port,
		config.Conf.Smtp.Account,
		config.Conf.Smtp.Password,
	)
	d.SSL = false //常见的 Smtp 端口：25（非加密）、465（SSL 加密）、587（TLS 加密） 这里根据配置文件来写
	return d.DialAndSend(m)
}
