package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"math/big"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"thuhole-go-backend/pkg/consts"
	"time"
)

func GenCode() string {
	nBig, err := rand.Int(rand.Reader, big.NewInt(100000000))
	if err != nil {
		panic(err)
	}
	n := nBig.Int64()
	return fmt.Sprintf("%08d", n)
}

func GenToken() string {
	randomBytes := make([]byte, 20)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}
	return strings.ToLower(base32.StdEncoding.EncodeToString(randomBytes))
}

func hash1(user string) string {
	h := sha256.New()
	h.Write([]byte(user))
	return hex.EncodeToString(h.Sum(nil))
}

func HashEmail(user string) string {
	return hash1(viper.GetString("salt") + hash1(user))
}

func GetTimeStamp() int64 {
	//loc, _ := time.LoadLocation("Asia/Shanghai")
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

func GetCommenterName(id int) string {
	switch {
	case id == 0:
		return consts.DzName
	case id <= 26:
		return consts.Names1[id-1]
	case id <= 26*27:
		return consts.Names0[(id-1)/26-1] + " " + consts.Names1[(id-1)%26]
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

func GetPinnedPids() []int {
	reg := regexp.MustCompile(`[ ,]`)
	s := viper.GetString("pinned_pids")
	var rtn []int
	for _, str := range reg.Split(s, -1) {
		if str != "" {
			i, err := strconv.Atoi(str)
			if err != nil {
				FatalErrorHandle(&err, "pinned_pids Atoi error:"+str)
			}
			rtn = append(rtn, i)
		}
	}
	return rtn
}

func GetReportWhitelistPids() []int {
	reg := regexp.MustCompile(`[ ,]`)
	s := viper.GetString("report_whitelist_pids")
	var rtn []int
	for _, str := range reg.Split(s, -1) {
		if str != "" {
			i, err := strconv.Atoi(str)
			if err != nil {
				FatalErrorHandle(&err, "report_whitelist_pids Atoi error:"+str)
			}
			rtn = append(rtn, i)
		}
	}
	return rtn
}

func CheckEmail(email string) bool {
	// REF: https://html.spec.whatwg.org/multipage/input.html#valid-e-mail-address
	var emailRegexp = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	return emailRegexp.MatchString(email)
}

func HttpReturnWithCodeOne(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"msg":  msg,
	})
}

func SafeSubSlice(slice []interface{}, low int, high int) []interface{} {
	if 0 <= low && low <= high && high <= cap(slice) {
		return slice[low:high]
	}
	return nil
}

var AllowedSubnets []*net.IPNet

func IsInSubnet(ip string) bool {
	for _, subnet := range AllowedSubnets {
		if subnet.Contains(net.ParseIP(ip)) {
			return true
		}
	}
	return false
}
