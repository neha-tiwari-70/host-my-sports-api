package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	config "sports-events-api/common"
	"strconv"
	"strings"
)

// var secretKey = []byte(os.Getenv("AES_KEY"))
// var secretKey = []byte("ea0d2f9fe37c180f44a2ac18ed9b9287")

func generateIV(plainText string) []byte {
	hash := md5.Sum([]byte(plainText))
	return hash[:aes.BlockSize]
}

func Encrypt(plainText string) (string, error) {

	plainTextBytes := []byte(plainText)
	block, err := aes.NewCipher(config.AESKey)
	if err != nil {
		return "", err
	}

	// gcm, err := cipher.NewGCM(block)
	// if err != nil {
	// 	return "", err
	// }

	// nonce := make([]byte, gcm.NonceSize())
	// _, err = io.ReadFull(rand.Reader, nonce)
	// if err != nil {
	// 	return "", err
	// }

	// cipherText := gcm.Seal(nonce, nonce, plainTextBytes, nil)
	cipherText := make([]byte, aes.BlockSize+len(plainTextBytes))
	iv := generateIV(plainText)

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], plainTextBytes)
	copy(cipherText[:aes.BlockSize], iv)

	return base64.RawURLEncoding.EncodeToString(cipherText), nil
}

func Decrypt(encryptedText string) (string, error) {
	cipherTextBytes, err := base64.RawURLEncoding.DecodeString(encryptedText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(config.AESKey)
	if err != nil {
		return "", err
	}

	// gcm, err := cipher.NewGCM(block)
	// if err != nil {
	// 	return "", err
	// }

	// nonceSize := gcm.NonceSize()
	// nonce, cipherText := cipherTextBytes[:nonceSize], cipherTextBytes[nonceSize:]

	// plainTextBytes, err := gcm.Open(nil, nonce, cipherText, nil)
	// if err != nil {
	// 	return "", err
	// }

	// return string(plainTextBytes), nil
	if len(cipherTextBytes) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	iv := cipherTextBytes[:aes.BlockSize]
	cipherTextBytes = cipherTextBytes[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherTextBytes, cipherTextBytes)
	return string(cipherTextBytes), nil
}
func NEncrypt(numberString int64) string {
	nestr := fmt.Sprintf("data-%d", numberString)
	str, err := Encrypt(nestr)
	if err != nil {
		panic(err)
	}
	return str
}

func NDecrypt(numberString string) (int64, error) {
	mainString, err := Decrypt(numberString)
	if err != nil {
		return 0, fmt.Errorf("invalid id")
	}
	substringToRemove := "data-"
	updatedString := strings.Replace(mainString, substringToRemove, "", -1)
	data, err := strconv.ParseInt(updatedString, 10, 64)
	if err != nil {
		return 0, err
	}
	return data, nil
}

type EncryptedString int64

func (bit *EncryptedString) UnmarshalJSON(data []byte) error {
	// Handle JSON null
	if string(data) == "null" {
		*bit = EncryptedString(0)
		return nil
	}

	asString := string(data)

	// Remove surrounding quotes (if any) using strconv.Unquote
	unquotedString, err := strconv.Unquote(asString)
	if err != nil {
		return errors.New("error unquoting string: " + err.Error())
	}

	if unquotedString == "" {
		*bit = EncryptedString(0)
		return nil
	}

	// Decrypt the unquoted string
	decryptedValue, err := NDecrypt(unquotedString)
	if err != nil {
		return err
	}
	*bit = EncryptedString(decryptedValue)
	return nil
}

func (bit EncryptedString) MarshalJSON() ([]byte, error) {

	if bit == 0 {
		return json.Marshal(nil)
	}

	encryptedString := NEncrypt(int64(bit))
	return json.Marshal(encryptedString)
}
