package mail

import (
	"bytes"
	"github.com/spf13/viper"
	"gopkg.in/gomail.v2"
	"html/template"
	"strconv"
)

func SendMail(code string, recipient string) error {
	websiteName := viper.GetString("name")
	m := gomail.NewMessage()
	m.SetHeader("From", viper.GetString("smtp_username"))
	m.SetHeader("To", recipient)
	m.SetHeader("Subject", "【"+websiteName+"】验证码")

	templateData := struct {
		Code  string
		Title string
	}{
		Code:  code,
		Title: websiteName,
	}

	port, err := strconv.Atoi(viper.GetString("smtp_port"))
	if err != nil {
		return err
	}
	t, err := template.ParseFiles("send_code.html")
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	if err = t.Execute(buf, templateData); err != nil {
		return err
	}
	m.SetBody("text/html", buf.String())
	m.AddAlternative("text/plain", "您好：\n\n欢迎您注册"+websiteName+"！\n\n"+code+"\n这是您注册"+websiteName+"的验证码，有效时间12小时。\n")
	d := gomail.NewDialer(viper.GetString("smtp_host"), port, viper.GetString("smtp_username"), viper.GetString("smtp_password"))

	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}
