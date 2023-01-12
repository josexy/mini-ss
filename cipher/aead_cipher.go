package cipher

import (
	"crypto/aes"
	"crypto/cipher"

	"golang.org/x/crypto/chacha20poly1305"
)

type AEADCipher interface {
	KeySize() int
	SaltSize() int
	Encrypter([]byte) (cipher.AEAD, error)
	Decrypter([]byte) (cipher.AEAD, error)
}

type metaAEADCipher struct {
	key      []byte
	saltSize int
	makeAEAD func(key []byte) (cipher.AEAD, error)
}

func (c *metaAEADCipher) KeySize() int {
	return len(c.key)
}

func (c *metaAEADCipher) SaltSize() int {
	return c.saltSize
}

func (c *metaAEADCipher) Encrypter(salt []byte) (cipher.AEAD, error) {
	outKey := make([]byte, c.KeySize())
	hkdfSha1(c.key, salt, []byte("ss-subkey"), outKey)
	return c.makeAEAD(outKey)
}

func (c *metaAEADCipher) Decrypter(salt []byte) (cipher.AEAD, error) {
	outKey := make([]byte, c.KeySize())
	hkdfSha1(c.key, salt, []byte("ss-subkey"), outKey)
	return c.makeAEAD(outKey)
}

func AESGCM(key []byte, saltSize int) (AEADCipher, error) {
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

func CHACHA20POLY1305(key []byte, saltSize int) (AEADCipher, error) {
	return &metaAEADCipher{
		key:      key,
		saltSize: saltSize,
		makeAEAD: chacha20poly1305.New,
	}, nil
}

func XCHACHA20POLY1305(key []byte, saltSize int) (AEADCipher, error) {
	return &metaAEADCipher{
		key:      key,
		saltSize: saltSize,
		makeAEAD: chacha20poly1305.NewX,
	}, nil
}
