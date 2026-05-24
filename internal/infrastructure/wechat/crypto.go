package wechat

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"
)

// MsgCrypt handles WeChat message encryption/decryption using AES-CBC.
type MsgCrypt struct {
	token          string
	encodingAESKey string
	appID          string
	aesKey         []byte
}

func NewMsgCrypt(token, encodingAESKey, appID string) (*MsgCrypt, error) {
	if len(encodingAESKey) != 43 {
		return nil, fmt.Errorf("wechat: invalid encoding_aes_key length %d, expected 43", len(encodingAESKey))
	}
	aesKey, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil {
		return nil, fmt.Errorf("wechat: decode aes key: %w", err)
	}
	return &MsgCrypt{
		token:          token,
		encodingAESKey: encodingAESKey,
		appID:          appID,
		aesKey:         aesKey,
	}, nil
}

// VerifySignature verifies the WeChat callback signature.
func (mc *MsgCrypt) VerifySignature(signature, timestamp, nonce string) bool {
	strs := []string{mc.token, timestamp, nonce}
	sort.Strings(strs)
	h := sha1.New()
	h.Write([]byte(strings.Join(strs, "")))
	expected := fmt.Sprintf("%x", h.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// VerifyEncryptedSignature verifies signature including encrypted message.
func (mc *MsgCrypt) VerifyEncryptedSignature(signature, timestamp, nonce, encrypt string) bool {
	strs := []string{mc.token, timestamp, nonce, encrypt}
	sort.Strings(strs)
	h := sha1.New()
	h.Write([]byte(strings.Join(strs, "")))
	expected := fmt.Sprintf("%x", h.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// DecryptMessage decrypts an AES-encrypted WeChat message.
func (mc *MsgCrypt) DecryptMessage(encryptedBase64 string) ([]byte, string, error) {
	cipherText, err := base64.StdEncoding.DecodeString(encryptedBase64)
	if err != nil {
		return nil, "", fmt.Errorf("wechat: base64 decode: %w", err)
	}

	block, err := aes.NewCipher(mc.aesKey)
	if err != nil {
		return nil, "", fmt.Errorf("wechat: new cipher: %w", err)
	}

	iv := mc.aesKey[:aes.BlockSize]
	mode := cipher.NewCBCDecrypter(block, iv)

	plainText := make([]byte, len(cipherText))
	mode.CryptBlocks(plainText, cipherText)
	plainText = pkcs7Unpad(plainText)

	// Format: 16-byte random + 4-byte msg_len (big-endian) + msg + appid
	if len(plainText) < 20 {
		return nil, "", fmt.Errorf("wechat: decrypted data too short")
	}

	msgLen := binary.BigEndian.Uint32(plainText[16:20])
	if int(msgLen)+20 > len(plainText) {
		return nil, "", fmt.Errorf("wechat: invalid message length")
	}

	msg := plainText[20 : 20+msgLen]
	appID := string(plainText[20+msgLen:])

	return msg, appID, nil
}

// EncryptMessage encrypts a message for WeChat reply.
func (mc *MsgCrypt) EncryptMessage(plainText []byte) (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, randomBytes); err != nil {
		return "", fmt.Errorf("wechat: gen random: %w", err)
	}

	msgLen := make([]byte, 4)
	binary.BigEndian.PutUint32(msgLen, uint32(len(plainText)))

	buf := bytes.NewBuffer(randomBytes)
	buf.Write(msgLen)
	buf.Write(plainText)
	buf.WriteString(mc.appID)

	padded := pkcs7Pad(buf.Bytes(), aes.BlockSize)

	block, err := aes.NewCipher(mc.aesKey)
	if err != nil {
		return "", fmt.Errorf("wechat: new cipher: %w", err)
	}

	iv := mc.aesKey[:aes.BlockSize]
	mode := cipher.NewCBCEncrypter(block, iv)

	cipherText := make([]byte, len(padded))
	mode.CryptBlocks(cipherText, padded)

	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// GenerateSignature generates a signature for encrypted reply.
func (mc *MsgCrypt) GenerateSignature(timestamp, nonce, encrypt string) string {
	strs := []string{mc.token, timestamp, nonce, encrypt}
	sort.Strings(strs)
	h := sha1.New()
	h.Write([]byte(strings.Join(strs, "")))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

func pkcs7Unpad(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	padding := int(data[len(data)-1])
	if padding < 1 || padding > len(data) || padding > aes.BlockSize {
		return data
	}
	for i := 0; i < padding; i++ {
		if data[len(data)-1-i] != byte(padding) {
			return data
		}
	}
	return data[:len(data)-padding]
}

// EncryptedXMLMessage is the WeChat encrypted message XML envelope.
type EncryptedXMLMessage struct {
	XMLName    xml.Name `xml:"xml"`
	Encrypt    string   `xml:"Encrypt"`
	MsgSig     string   `xml:"MsgSignature"`
	TimeStamp  string   `xml:"TimeStamp"`
	Nonce      string   `xml:"Nonce"`
}

// EncryptedReply formats an encrypted reply for WeChat.
type EncryptedReply struct {
	XMLName    xml.Name `xml:"xml"`
	Encrypt    string   `xml:"Encrypt"`
	MsgSig     string   `xml:"MsgSignature"`
	TimeStamp  string   `xml:"TimeStamp"`
	Nonce      string   `xml:"Nonce"`
}
