package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"treehollow-v3-backend/pkg/consts"
)

func Pad(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padText...)
}

func Unpad(src []byte) ([]byte, error) {
	length := len(src)
	unpadding := int(src[length-1])

	if unpadding > length {
		return nil, errors.New("unpad error. This could happen when incorrect encryption key is used")
	}

	return src[:(length - unpadding)], nil
}

//See https://gist.github.com/thuhole/ba41ade1ca97be838ddfcb030306d997
func AESEncrypt(plaintext string, keyStr string) (string, error) {
	h := sha256.New()
	h.Write([]byte(keyStr))
	key := h.Sum(nil)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()

	msg := Pad([]byte(plaintext), blockSize)
	ciphertext := make([]byte, blockSize+len(msg))
	iv := ciphertext[:blockSize]
	if _, err = io.ReadFull(bytes.NewReader([]byte(consts.AesIv)), iv); err != nil {
		return "", err
	}

	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[blockSize:], msg)
	finalMsg := hex.EncodeToString(ciphertext)
	return finalMsg, nil
}

func AESDecrypt(ciphertext string, keyStr string) (string, error) {
	h := sha256.New()
	h.Write([]byte(keyStr))
	key := h.Sum(nil)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()

	decodedMsg, err := hex.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	if (len(decodedMsg) % blockSize) != 0 {
		return "", errors.New("block_size must be multiple of decoded message length")
	}

	iv := decodedMsg[:blockSize]
	msg := decodedMsg[blockSize:]

	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(msg, msg)

	unpadMsg, err := Unpad(msg)
	if err != nil {
		return "", err
	}

	return string(unpadMsg), nil
}
