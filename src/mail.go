package main

import (
	"bytes"
	"context"
	"github.com/mailgun/mailgun-go"
	"github.com/spf13/viper"
	"gopkg.in/gomail.v2"
	"html/template"
	"time"
)

func sendMail(code string, recipient string) (string, error) {
	apiKey := viper.GetString("mailgun_key")
	domain := viper.GetString("mailgun_domain")
	mg := mailgun.NewMailgun(domain, apiKey)
	m := mg.NewMessage(
		"T大树洞 <noreply@"+domain+">",
		"【T大树洞】验证码",
		"您好：\n\n欢迎您注册T大树洞！\n\n"+code+"\n这是您注册T大树洞的验证码，有效时间12小时。\n",
		recipient,
	)
	m.SetTemplate("code")
	_ = m.AddVariable("code", code)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	_, id, err := mg.Send(ctx, m)
	return id, err
}

func sendMail2(code string, recipient string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", viper.GetString("smtp_username"))
	m.SetHeader("To", recipient)
	m.SetHeader("Subject", "【T大树洞】验证码")

	templateData := struct {
		Code string
	}{
		Code: code,
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
	m.AddAlternative("text/plain", "您好：\n\n欢迎您注册T大树洞！\n\n"+code+"\n这是您注册T大树洞的验证码，有效时间12小时。\n")
	d := gomail.NewDialer(viper.GetString("smtp_host"), 465, viper.GetString("smtp_username"), viper.GetString("smtp_password"))

	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}
