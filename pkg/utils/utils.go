package utils

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/google/uuid"
	errors2 "github.com/pkg/errors"
	"github.com/sigurn/crc8"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"treehollow-v3-backend/pkg/consts"
	"treehollow-v3-backend/pkg/logger"
)

var AllowedSubnets []*net.IPNet
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

func GenNonce() string {
	return uuid.New().String()
}

func SHA256(user string) string {
	h := sha256.New()
	h.Write([]byte(user))
	return hex.EncodeToString(h.Sum(nil))
}

func HashEmail(user string) string {
	return SHA256(Salt + SHA256(strings.ToLower(user)))
}

func GetTimeStamp() int64 {
	return time.Now().Unix()
}

func FatalErrorHandle(err *error, msg string) {
	if *err != nil {
		panic(fmt.Errorf("Fatal error: %s \n %s \n", msg, *err))
	}
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

func ContainsInt(s []int, e int) (int, bool) {
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

func CheckEmail(email string) bool {
	// REF: https://html.spec.whatwg.org/multipage/input.html#valid-e-mail-address
	var emailRegexp = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	return emailRegexp.MatchString(email)
}

func CreatePublicKeyRing(publicKey string) (*crypto.KeyRing, error) {
	publicKeyObj, err := crypto.NewKeyFromArmored(publicKey)
	if err != nil {
		return nil, errors2.Wrap(err, "gopenpgp: unable to parse public key")
	}

	if publicKeyObj.IsPrivate() {
		publicKeyObj, err = publicKeyObj.ToPublic()
		if err != nil {
			return nil, errors2.Wrap(err, "gopenpgp: unable to extract public key from private key")
		}
	}

	publicKeyRing, err := crypto.NewKeyRing(publicKeyObj)
	if err != nil {
		return nil, errors2.Wrap(err, "gopenpgp: unable to create new keyring")
	}

	return publicKeyRing, nil
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

func SaveImage(base64img string, imgPath string) ([]byte, string, string, *logger.InternalError) {
	var suffix string
	sDec, err2 := base64.StdEncoding.DecodeString(base64img)
	if err2 != nil {
		return nil, "", "{}", logger.NewSimpleError("InvalidImgBase64", "图片数据不合法", logger.WARN)
	}
	fileType := http.DetectContentType(sDec)
	if fileType != "image/jpeg" && fileType != "image/jpg" && fileType != "image/png" && fileType != "image/gif" {
		return nil, "", "{}", logger.NewSimpleError("InvalidImgType", "图片数据不合法", logger.WARN)
	}

	im, _, err := image.DecodeConfig(bytes.NewReader(sDec))
	if err != nil {
		return nil, "", "{}", logger.NewError(err, "ImageDecodeFailed", "图片解析失败")
	}
	if im.Width > consts.ImageMaxWidth || im.Height > consts.ImageMaxHeight {
		return nil, "", "{}", logger.NewSimpleError("TooLargeImg", "图片过大", logger.WARN)
	}
	metadataBytes, err := json.Marshal(map[string]int{"w": im.Width, "h": im.Height})
	if err != nil {
		return nil, "", "{}", logger.NewError(err, "ImageSizeDecodeFailed", "图片大小解析失败")
	}

	if fileType == "image/png" {
		suffix = ".png"
	} else if fileType == "image/gif" {
		suffix = ".gif"
	} else {
		suffix = ".jpeg"
	}

	hashedPath := filepath.Join(viper.GetString("images_path"), imgPath[:2])
	_ = os.MkdirAll(hashedPath, os.ModePerm)
	err3 := ioutil.WriteFile(filepath.Join(hashedPath, imgPath+suffix), sDec, 0644)
	if err3 != nil {
		return nil, suffix, string(metadataBytes), logger.NewError(err3, "ErrorSavingImage", "图片存储失败")
	}
	return sDec, suffix, string(metadataBytes), nil
}

func CalcExtra(str1 string, str2 string) int64 {
	table := crc8.MakeTable(crc8.CRC8)
	rtn := int64(crc8.Checksum([]byte(str2+str1), table) % 4)

	return rtn
}

type void struct{}

var member void

func Int32SliceToSet(ids []int32) map[int32]void {
	set := make(map[int32]void)
	for _, id := range ids {
		set[id] = member
	}
	return set
}

func Int32IsInSet(id int32, ids map[int32]void) (rtn bool) {
	_, rtn = ids[id]
	return
}

func TimestampToString(timestamp int64) string {
	return time.Unix(timestamp, 0).
		In(consts.TimeLoc).Format("01-02 15:04")
}

func TrimText(text string, maxLength int) string {
	runeStr := []rune(text)
	if len(runeStr) > maxLength {
		return string(runeStr[:maxLength]) + "..."
	}
	return text
}

func GetEarliestAuthenticationTime() time.Time {
	return time.Now().AddDate(0, 0, -consts.TokenExpireDays)
}

func UnscopedTx(tx *gorm.DB, b bool) *gorm.DB {
	if b {
		return tx.Unscoped()
	}
	return tx
}
