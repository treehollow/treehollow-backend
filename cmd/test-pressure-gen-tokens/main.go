package main

import (
	"fmt"
	"github.com/google/uuid"
	"os"
	"treehollow-v3-backend/pkg/base"
	"treehollow-v3-backend/pkg/config"
	"treehollow-v3-backend/pkg/utils"
)

const N = 10000

func main() {
	config.InitConfigFile()
	base.InitDb()

	logFile, err := os.OpenFile("pressure_test_tokens.txt", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()

	user := base.User{
		EmailEncrypted: "PressureTestUser",
		ForgetPwNonce:  utils.GenNonce(),
		Role:           base.NormalUserRole,
	}
	if err = base.GetDb(false).Create(&user).Error; err != nil {
		panic(err)
	}

	var devices = make([]base.Device, 0, N)
	for i := 0; i < N; i++ {
		token := utils.GenToken()
		_, _ = fmt.Fprintln(logFile, token)
		devices = append(devices, base.Device{
			ID:             uuid.New().String(),
			UserID:         user.ID,
			Token:          token,
			DeviceInfo:     "PressureTestToken",
			Type:           base.AndroidDevice,
			LoginIP:        "127.0.0.1",
			LoginCity:      "Unknown",
			IOSDeviceToken: "",
		})
	}

	if err = base.GetDb(false).CreateInBatches(&devices, 1000).Error; err != nil {
		panic(err)
	}
}
