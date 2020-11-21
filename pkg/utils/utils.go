package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	libredis "github.com/go-redis/redis/v7"
	"github.com/oschwald/geoip2-golang"
	"github.com/sigurn/crc8"
	"github.com/spf13/viper"
	"github.com/ulule/limiter/v3"
	sredis "github.com/ulule/limiter/v3/drivers/store/redis"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"thuhole-go-backend/pkg/consts"
	"time"
)

var AllowedSubnets []*net.IPNet
var GeoDb *geoip2.Reader
var Salt string

func GenCode() string {
	nBig, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		panic(err)
	}
	n := nBig.Int64()
	return fmt.Sprintf("%06d", n)
}

func GenToken() string {
	randomBytes := make([]byte, 20)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}
	return strings.ToLower(base32.StdEncoding.EncodeToString(randomBytes))
}

func Hash1(user string) string {
	h := sha256.New()
	h.Write([]byte(user))
	return hex.EncodeToString(h.Sum(nil))
}

func HashEmail(user string) string {
	return Hash1(Salt + Hash1(strings.ToLower(user)))
}

func GetTimeStamp() int64 {
	return time.Now().Unix()
}

func FatalErrorHandle(err *error, msg string) {
	if *err != nil {
		panic(fmt.Errorf("Fatal error: %s \n %s \n", msg, *err))
	}
}

func ContainsInt(s []int, e int) (int, bool) {
	i := -1
	for i, a := range s {
		if a == e {
			return i, true
		}
	}
	return i, false
}

func ContainsString(s []string, e string) (int, bool) {
	i := -1
	for i, a := range s {
		if a == e {
			return i, true
		}
	}
	return i, false
}

func GetCommenterName(id int, names0 []string, names1 []string) string {
	switch {
	case id == 0:
		return consts.DzName
	case id <= 26:
		return names1[id-1]
	case id <= 26*27:
		return names0[(id-1)/26-1] + " " + names1[(id-1)%26]
	default:
		return consts.ExtraNamePrefix + strconv.Itoa(id-26*27)
	}
}

//func remove(s []int, i int) []int {
//	s[len(s)-1], s[i] = s[i], s[len(s)-1]
//	return s[:len(s)-1]
//}

func IfThenElse(condition bool, a interface{}, b interface{}) interface{} {
	if condition {
		return a
	}
	return b
}

func SplitToString(a []int, sep string) string {
	if len(a) == 0 {
		return ""
	}

	b := make([]string, len(a))
	for i, v := range a {
		b[i] = strconv.Itoa(v)
	}
	return strings.Join(b, sep)
}

func CheckEmail(email string) bool {
	// REF: https://html.spec.whatwg.org/multipage/input.html#valid-e-mail-address
	var emailRegexp = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	return emailRegexp.MatchString(email)
}

func HttpReturnWithCodeOne(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"msg":  msg,
	})
}

func SafeSubSlice(slice []gin.H, low int, high int) []gin.H {
	if 0 <= low && low <= high && high <= len(slice) {
		return slice[low:high]
	}
	return nil
}

func IsInAllowedSubnet(ip string) bool {
	for _, subnet := range AllowedSubnets {
		if subnet.Contains(net.ParseIP(ip)) {
			return true
		}
	}
	return false
}

func GetHashedFilePath(filePath string) string {
	if len(filePath) > 2 {
		return filePath[:2] + "/" + filePath
	}
	return filePath
}

func InitLimiter(rate limiter.Rate, prefix string) *limiter.Limiter {
	option, err := libredis.ParseURL(viper.GetString("redis_source"))
	if err != nil {
		FatalErrorHandle(&err, "failed init redis url")
		return nil
	}
	client := libredis.NewClient(option)
	store, err2 := sredis.NewStoreWithOptions(client, limiter.StoreOptions{
		Prefix:   prefix,
		MaxRetry: 3,
	})
	if err2 != nil {
		FatalErrorHandle(&err2, "failed init redis store")
		return nil
	}
	return limiter.New(store, rate)
}

func SaveImage(base64img string, imgPath string) ([]byte, string, error) {
	var suffix string
	sDec, err2 := base64.StdEncoding.DecodeString(base64img)
	if err2 != nil {
		return nil, "", errors.New("图片数据不合法")
	}
	fileType := http.DetectContentType(sDec)
	if fileType != "image/jpeg" && fileType != "image/jpg" && fileType != "image/png" {
		return nil, "", errors.New("图片数据不合法")
	}

	if fileType == "image/png" {
		suffix = ".png"
	} else {
		suffix = ".jpeg"
	}

	hashedPath := filepath.Join(viper.GetString("images_path"), imgPath[:2])
	_ = os.MkdirAll(hashedPath, os.ModePerm)
	err3 := ioutil.WriteFile(filepath.Join(hashedPath, imgPath+suffix), sDec, 0644)
	if err3 != nil {
		log.Printf("error ioutil.WriteFile while saving image, err=%s\n", err3.Error())
		return nil, suffix, errors.New("error while saving image")
	}
	return sDec, suffix, nil
}

func CalcExtra(str1 string, str2 string) int {
	table := crc8.MakeTable(crc8.CRC8)
	rtn := int(crc8.Checksum([]byte(str2+str1), table) % 4)

	return rtn
}

func ProcessExtra(data []gin.H, str string, keyStr string) {
	for _, item := range data {
		item["timestamp"] = item["timestamp"].(int) - CalcExtra(str, strconv.Itoa(item[keyStr].(int)))
	}
}
