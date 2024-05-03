package cipher

import (
	"crypto/aes"
	"crypto/cipher"

	"golang.org/x/crypto/chacha20poly1305"
)

var info = []byte("ss-subkey")

type AEADCipher interface {
	KeySize() int
	SaltSize() int
	GetEncrypter([]byte) (cipher.AEAD, error)
	GetDecrypter([]byte) (cipher.AEAD, error)
}

type metaAEADCipher struct {
	key      []byte
	saltSize int
	makeAEAD func(key []byte) (cipher.AEAD, error)
}

func (c *metaAEADCipher) KeySize() int { return len(c.key) }

func (c *metaAEADCipher) SaltSize() int { return c.saltSize }

func (c *metaAEADCipher) GetEncrypter(salt []byte) (cipher.AEAD, error) {
	outKey := make([]byte, c.KeySize())
	hkdfSha1(c.key, salt, info, outKey)
	return c.makeAEAD(outKey)
}

func (c *metaAEADCipher) GetDecrypter(salt []byte) (cipher.AEAD, error) {
	outKey := make([]byte, c.KeySize())
	hkdfSha1(c.key, salt, info, outKey)
	return c.makeAEAD(outKey)
}

func AesGcm(key []byte, saltSize int) (AEADCipher, error) {
	return &metaAEADCipher{
		key:      key,
		saltSize: saltSize,
		makeAEAD: func(key []byte) (cipher.AEAD, error) {
			blk, err := aes.NewCipher(key)
			if err != nil {
				return nil, err
			}
			return cipher.NewGCM(blk)
		},
	}, nil
}

func Chacha20Poly1305(key []byte, saltSize int) (AEADCipher, error) {
	return &metaAEADCipher{
		key:      key,
		saltSize: saltSize,
		makeAEAD: chacha20poly1305.New,
	}, nil
}

func XChacha20Poly1305(key []byte, saltSize int) (AEADCipher, error) {
	return &metaAEADCipher{
		key:      key,
		saltSize: saltSize,
		makeAEAD: chacha20poly1305.NewX,
	}, nil
}
