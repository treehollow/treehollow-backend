package main

import (
	"context"
	"github.com/mailgun/mailgun-go"
	"github.com/spf13/viper"
	"time"
)

func sendMail(code string, recipient string) (string, error) {
	apiKey := viper.GetString("mailgun_key")
	domain := viper.GetString("mailgun_domain")
	mg := mailgun.NewMailgun(domain, apiKey)
	m := mg.NewMessage(
		"T大树洞 <noreply@"+domain+">",
		"【T大树洞验证码】"+code,
		"您好：\n\n欢迎您注册T大树洞！\n\n"+code+"\n这是您注册T大树洞的验证码，有效时间15分钟。\n",
		recipient,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	_, id, err := mg.Send(ctx, m)
	return id, err
}
