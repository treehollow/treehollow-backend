package s3

import (
	"os"
	"strings"
	"testing"
	"treehollow-v3-backend/pkg/config"
)

func TestS3(t *testing.T) {
	if os.Getenv("TRAVIS") != "true" {
		_ = os.Chdir("..")
		_ = os.Chdir("..")
		config.InitConfigFile()
		err := Upload("test/test.txt", strings.NewReader("hello1"))
		if err != nil {
			t.Errorf("err=%s", err)
		}
	}
}
