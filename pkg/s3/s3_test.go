package s3

import (
	"os"
	"strings"
	"testing"
	"thuhole-go-backend/pkg/config"
)

func TestS3(t *testing.T) {
	_ = os.Chdir("..")
	_ = os.Chdir("..")
	config.InitConfigFile()
	err := Upload("test/test.txt", strings.NewReader("hello1"))
	if err != nil {
		t.Errorf("err=%s", err)
	}
}
