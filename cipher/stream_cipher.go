package cipher

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rc4"

	"golang.org/x/crypto/blowfish"
	"golang.org/x/crypto/chacha20"
	"golang.org/x/crypto/salsa20"
)

type StreamCipher interface {
	KeySize() int
	IVSize() int
	Key() []byte
	GetEncrypter([]byte) (cipher.Stream, error)
	GetDecrypter([]byte) (cipher.Stream, error)
}

type metaStreamCipher struct {
	key           []byte
	ivSize        int
	makeEncrypter func(key, iv []byte) (cipher.Stream, error)
	makeDecrypter func(key, iv []byte) (cipher.Stream, error)
}

func (m *metaStreamCipher) Key() []byte { return m.key }

func (m *metaStreamCipher) KeySize() int { return len(m.key) }

func (m *metaStreamCipher) IVSize() int { return m.ivSize }

func (m *metaStreamCipher) GetEncrypter(iv []byte) (cipher.Stream, error) {
	return m.makeEncrypter(m.key, iv)
}

func (m *metaStreamCipher) GetDecrypter(iv []byte) (cipher.Stream, error) {
	return m.makeDecrypter(m.key, iv)
}

func aesCTR(key, iv []byte) (cipher.Stream, error) {
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewCTR(blk, iv), nil
}

func AesCtr(key []byte, ivSize int) (StreamCipher, error) {
	// nopadding
	return &metaStreamCipher{
		key:           key,
		ivSize:        ivSize,
		makeEncrypter: aesCTR,
		makeDecrypter: aesCTR,
	}, nil
}

func AesCfb(key []byte, ivSize int) (StreamCipher, error) {
	// nopadding
	return &metaStreamCipher{
		key:    key,
		ivSize: ivSize,
		makeEncrypter: func(key, iv []byte) (cipher.Stream, error) {
			blk, err := aes.NewCipher(key)
			if err != nil {
				return nil, err
			}
			return cipher.NewCFBEncrypter(blk, iv), nil
		},
		makeDecrypter: func(key, iv []byte) (cipher.Stream, error) {
			blk, err := aes.NewCipher(key)
			if err != nil {
				return nil, err
			}
			return cipher.NewCFBDecrypter(blk, iv), nil
		},
	}, nil
}

func BfCfb(key []byte, ivSize int) (StreamCipher, error) {
	return &metaStreamCipher{
		key:    key,
		ivSize: ivSize,
		makeEncrypter: func(key, iv []byte) (cipher.Stream, error) {
			blk, err := blowfish.NewCipher(key)
			if err != nil {
				return nil, err
			}
			return cipher.NewCFBEncrypter(blk, iv), nil
		},
		makeDecrypter: func(key, iv []byte) (cipher.Stream, error) {
			blk, err := blowfish.NewCipher(key)
			if err != nil {
				return nil, err
			}
			return cipher.NewCFBDecrypter(blk, iv), nil
		},
	}, nil
}

func rc4MD5(key, iv []byte) (cipher.Stream, error) {
	h := md5.New()
	h.Write(key)
	h.Write(iv)
	rc4key := h.Sum(nil)
	return rc4.NewCipher(rc4key)
}

func Rc4Md5(key []byte, ivSize int) (StreamCipher, error) {
	return &metaStreamCipher{
		key:           key,
		ivSize:        ivSize,
		makeEncrypter: rc4MD5,
		makeDecrypter: rc4MD5,
	}, nil
}

// salsa20 cipher wrapper
type salsa20Wrapper struct {
	key   []byte // 32 bytes
	nonce []byte // 8 bytes
}

func newSalsa20(key, iv []byte) (cipher.Stream, error) {
	return &salsa20Wrapper{
		key:   key,
		nonce: iv,
	}, nil
}

func (s *salsa20Wrapper) XORKeyStream(dst, src []byte) {
	salsa20.XORKeyStream(dst, src, s.nonce[:], (*[32]byte)(s.key[:32]))
}

func Salsa20(key []byte, ivSize int) (StreamCipher, error) {
	return &metaStreamCipher{
		key:           key,
		ivSize:        ivSize,
		makeEncrypter: newSalsa20,
		makeDecrypter: newSalsa20,
	}, nil
}

// chacha20 cipher wrapper
type chacha20Wrapper struct {
	cp *chacha20.Cipher
}

func newChacha20(key, iv []byte) (cipher.Stream, error) {
	cp, err := chacha20.NewUnauthenticatedCipher(key, iv)
	if err != nil {
		return nil, err
	}
	return &chacha20Wrapper{cp: cp}, nil
}

func (c *chacha20Wrapper) XORKeyStream(dst, src []byte) {
	c.cp.XORKeyStream(dst, src)
}

func Chacha20(key []byte, ivSize int) (StreamCipher, error) {
	return &metaStreamCipher{
		key:           key,
		ivSize:        ivSize,
		makeEncrypter: newChacha20,
		makeDecrypter: newChacha20,
	}, nil
}

func Chacha20Ietf(key []byte, ivSize int) (StreamCipher, error) {
	return Chacha20(key, ivSize)
}
