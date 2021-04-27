package utils

import (
	"fmt"
	"testing"
)

var (
	aesTestCases = []struct {
		plainText string
		key       string
	}{
		{plainText: "1", key: "3"},
		{plainText: "2", key: "4"},
		{plainText: "3", key: ""},
		{plainText: "1111111111111111111111111111111111111111111111111111111111111111111111111123",
			key: "1111111111111111111111111111111111111111111111111111111111111111111111111123"},
	}
)

func TestAes(t *testing.T) {
	for _, c := range aesTestCases {
		fmt.Println("plaintext length:", len(c.plainText))
		fmt.Println("key length:", len(c.key))
		cipherText, err := AESEncrypt(c.plainText, c.key)
		if err != nil {
			t.Errorf(err.Error())
		}
		cipherText2, err := AESEncrypt(c.plainText, c.key)
		if err != nil {
			t.Errorf(err.Error())
		}
		if cipherText2 != cipherText {
			t.Errorf("Encryption is random!")
		}
		fmt.Println("ciphertext length:", len(cipherText))
		fmt.Println()

		newPlainText, err2 := AESDecrypt(cipherText, c.key)
		if err2 != nil {
			t.Errorf(err2.Error())
		}
		if newPlainText != c.plainText {
			t.Errorf("Decrypted text does not match!")
		}
	}
}
