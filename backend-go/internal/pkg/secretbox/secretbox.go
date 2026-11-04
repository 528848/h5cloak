// Package secretbox 用 AES-256-GCM 加密需要可恢复的密钥（站点服务端密钥用于 HMAC 验签）。
// 主密钥来自配置 auth.secret_key；为空时从 jwt_secret 派生。
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
)

// Box 加解密器。
type Box struct{ aead cipher.AEAD }

// New 用任意长度的主密钥创建（内部 SHA-256 派生 32 字节）。
func New(master string) (*Box, error) {
	if master == "" {
		return nil, errors.New("secretbox: empty master key")
	}
	key := sha256.Sum256([]byte(master))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Box{aead: aead}, nil
}

// Seal 加密，输出 nonce||ciphertext。
func (b *Box) Seal(plain []byte) ([]byte, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return append(nonce, b.aead.Seal(nil, nonce, plain, nil)...), nil
}

// Open 解密。
func (b *Box) Open(data []byte) ([]byte, error) {
	n := b.aead.NonceSize()
	if len(data) < n {
		return nil, errors.New("secretbox: ciphertext too short")
	}
	return b.aead.Open(nil, data[:n], data[n:], nil)
}
