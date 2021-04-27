package s3

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/spf13/viper"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func DogeCloudAPI(apiPath string, data map[string]interface{}, jsonMode bool) (ret map[string]interface{}, err error) {
	AccessKey := viper.GetString("DCAccessKey")
	SecretKey := viper.GetString("DCSecretKey")

	body := ""
	mime := ""
	if jsonMode {
		_body, err := json.Marshal(data)
		if err != nil {
			log.Fatalln(err)
		}
		body = string(_body)
		mime = "application/json"
	} else {
		values := url.Values{}
		for k, v := range data {
			values.Set(k, v.(string))
		}
		body = values.Encode()
		mime = "application/x-www-form-urlencoded"
	}

	signStr := apiPath + "\n" + body
	hmacObj := hmac.New(sha1.New, []byte(SecretKey))
	hmacObj.Write([]byte(signStr))
	sign := hex.EncodeToString(hmacObj.Sum(nil))
	Authorization := "TOKEN " + AccessKey + ":" + sign

	req, err := http.NewRequest("POST", "https://api.dogecloud.com"+apiPath, strings.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Add("Content-Type", mime)
	req.Header.Add("Authorization", Authorization)
	client := http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	} // 网络错误
	defer resp.Body.Close()
	r, _ := ioutil.ReadAll(resp.Body)

	_ = json.Unmarshal(r, &ret)

	fmt.Printf("[DogeCloudAPI] code: %d, msg: %s, data: %s\n", int(ret["code"].(float64)), ret["msg"], ret["data"])
	return
}

func Upload(filePath string, fileReader io.ReadSeeker) error {
	prof := make(map[string]interface{})
	prof["channel"] = "OSS_FULL"
	prof["scopes"] = "*"
	r, err := DogeCloudAPI("/auth/tmp_token.json", prof, true)
	if err != nil {
		return err
	}
	if r["data"] == nil {
		return errors.New("invalid DogeCloud response")
	}
	data := r["data"].(map[string]interface{})
	creds := data["Credentials"].(map[string]interface{})

	s3Config := &aws.Config{
		Credentials: credentials.NewStaticCredentials(creds["accessKeyId"].(string), creds["secretAccessKey"].(string), creds["sessionToken"].(string)),
		Region:      aws.String("automatic"),
		Endpoint:    aws.String(viper.GetString("DCS3Endpoint")),
	}

	newSession, err := session.NewSession(s3Config)
	if err != nil {
		return err
	}

	s3Client := s3.New(newSession)

	_, err = s3Client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(viper.GetString("DCS3Bucket")),
		Key:    aws.String(filePath),
		Body:   fileReader,
	})
	return err
}
