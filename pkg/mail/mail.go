package mail

import (
	"github.com/spf13/viper"
	"gopkg.in/gomail.v2"
	"strconv"
)

func SendValidationEmail(code string, recipient string) error {
	websiteName := viper.GetString("name")
	m := gomail.NewMessage()
	m.SetHeader("From", viper.GetString("smtp_username"))
	m.SetHeader("To", recipient)
	title := "【" + websiteName + "】验证码"
	m.SetHeader("Subject", title)

	msg := `<!DOCTYPE html>
<html lang="cn">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + title + `</title>
</head>
<body>
<p>欢迎您注册` + websiteName + `！</p>
<p>这是您的验证码，有效时间12小时。</p>
<p><strong>` + code + `</strong></p>
</body>
</html>`

	port, err := strconv.Atoi(viper.GetString("smtp_port"))
	if err != nil {
		return err
	}
	m.SetBody("text/html", msg)
	m.AddAlternative("text/plain", "您好：\n\n欢迎您注册"+websiteName+"！\n\n"+code+"\n这是您注册"+websiteName+"的验证码，有效时间12小时。\n")
	d := gomail.NewDialer(viper.GetString("smtp_host"), port, viper.GetString("smtp_username"), viper.GetString("smtp_password"))

	if err = d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}

func SendUnregisterValidationEmail(code string, recipient string) error {
	websiteName := viper.GetString("name")
	m := gomail.NewMessage()
	m.SetHeader("From", viper.GetString("smtp_username"))
	m.SetHeader("To", recipient)
	title := "【" + websiteName + "】验证码"
	m.SetHeader("Subject", title)

	msg := `<!DOCTYPE html>
<html lang="cn">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + title + `</title>
</head>
<body>
<p>您好，您正在注销` + websiteName + `。</p>
<p>这是您的验证码，有效时间12小时。</p>
<p><strong>` + code + `</strong></p>
</body>
</html>`

	port, err := strconv.Atoi(viper.GetString("smtp_port"))
	if err != nil {
		return err
	}
	m.SetBody("text/html", msg)
	m.AddAlternative("text/plain", "您好：\n\n您好，您正在注销"+websiteName+"。\n\n"+code+"\n这是您的验证码，有效时间12小时。\n")
	d := gomail.NewDialer(viper.GetString("smtp_host"), port, viper.GetString("smtp_username"), viper.GetString("smtp_password"))

	if err = d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}

func SendPasswordNonceEmail(nonce string, recipient string) error {
	websiteName := viper.GetString("name")
	m := gomail.NewMessage()
	m.SetHeader("From", viper.GetString("smtp_username"))
	m.SetHeader("To", recipient)
	title := "欢迎您注册" + websiteName
	m.SetHeader("Subject", title)

	msg := `<!DOCTYPE html>
<html lang="cn">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + title + `</title>
</head>
<body>
<p>欢迎您注册` + websiteName + `！</p>
<p>下方的字符串是当您忘记密码时可以帮助您找回密码的口令，请您妥善保管。</p>
<p><strong>` + nonce + `</strong></p>
</body>
</html>`

	port, err := strconv.Atoi(viper.GetString("smtp_port"))
	if err != nil {
		return err
	}
	m.SetBody("text/html", msg)
	m.AddAlternative("text/plain", "您好：\n\n欢迎您注册"+websiteName+"！\n下方的字符串是当您忘记密码时可以帮助您找回密码的口令，请您妥善保管。\n"+nonce+"\n")
	d := gomail.NewDialer(viper.GetString("smtp_host"), port, viper.GetString("smtp_username"), viper.GetString("smtp_password"))

	if err = d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}
