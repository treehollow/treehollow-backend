package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
)

func genCode() string {
	nBig, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		panic(err)
	}
	n := nBig.Int64()
	return fmt.Sprintf("%06d", n)
}

func genToken() string {
	randomBytes := make([]byte, 20)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}
	return strings.ToLower(base32.StdEncoding.EncodeToString(randomBytes))
}

func hashEmail(user string) string {
	h := sha256.New()
	h.Write([]byte(user))
	return hex.EncodeToString(h.Sum(nil))
}

func getTimeStamp() int64 {
	//loc, _ := time.LoadLocation("Asia/Shanghai")
	return time.Now().Unix()
}

func fatalErrorHandle(err *error, msg string) {
	if *err != nil {
		panic(fmt.Errorf("Fatal error: %s \n %s \n", msg, *err))
	}
}

func charToInt(c int) int {
	if c <= '9' {
		return c - int('0')
	} else {
		return c - int('a') + 10
	}
}

func hexToIntSlice(str string) []int {
	rtn := make([]int, len(str)/8)

	res := int(0)
	for i, r := range str {
		res = res + charToInt(int(r))<<(4*(7-(i%8)))
		if (i+1)%8 == 0 {
			rtn[i/8] = res
			res = 0
		}
	}
	return rtn
}

func intSliceToHex(array []int) string {
	var rtn string
	for _, n := range array {
		rtn += fmt.Sprintf("%08x", n)
	}
	return rtn
}

func contains(s []int, e int) (int, bool) {
	i := -1
	for i, a := range s {
		if a == e {
			return i, true
		}
	}
	return i, false
}

func getCommenterName(id int) string {
	switch {
	case id == 0:
		return dzName
	case id <= 26:
		return names1[id-1]
	case id <= 26*27:
		return names0[(id-1)/26-1] + " " + names1[(id-1)%26]
	default:
		return extraNamePrefix + strconv.Itoa(id-26*27)
	}
}

func remove(s []int, i int) []int {
	s[len(s)-1], s[i] = s[i], s[len(s)-1]
	return s[:len(s)-1]
}

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
