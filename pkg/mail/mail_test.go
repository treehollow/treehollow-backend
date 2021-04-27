package mail

import (
	"os"
	"testing"
	"treehollow-v3-backend/pkg/config"
)

func TestSendCode(t *testing.T) {
	_ = os.Chdir("..")
	_ = os.Chdir("..")
	config.InitConfigFile()
	err := SendValidationEmail("123456", "test-treehollow3@srv1.mail-tester.com")
	if err != nil {
		t.Errorf("error: %s", err)
	}
}

func TestSendNonce(t *testing.T) {
	_ = os.Chdir("..")
	_ = os.Chdir("..")
	config.InitConfigFile()
	err := SendPasswordNonceEmail("nonce-198247832648712631", "test-treehollow3@srv1.mail-tester.com")
	if err != nil {
		t.Errorf("error: %s", err)
	}
}
