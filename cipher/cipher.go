package cipher

import (
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"io"

	"github.com/josexy/mini-ss/util/logger"
	"golang.org/x/crypto/hkdf"
)

const (
	aes128CTR    = "aes-128-ctr"
	aes192CTR    = "aes-192-ctr"
	aes256CTR    = "aes-256-ctr"
	aes128CFB    = "aes-128-cfb"
	aes192CFB    = "aes-192-cfb"
	aes256CFB    = "aes-256-cfb"
	bfCFB        = "bf-cfb" // Blowfish in CFB mode
	salsa20_     = "salsa20"
	rc4Md5       = "rc4-md5"
	chacha20_    = "chacha20"
	chacha20IETF = "chacha20-ietf"
)

const (
	aes128GCM         = "aes-128-gcm"
	aes192GCM         = "aes-192-gcm"
	aes256GCM         = "aes-256-gcm"
	chacha20Poly1305  = "chacha20-ietf-poly1305"
	xchacha20Poly1305 = "xchacha20-ietf-poly1305"
)

var (
	streamCipherMap = map[string]streamCipherWrapper{
		aes128CTR:    {16, 16, AesCtr},
		aes192CTR:    {24, 16, AesCtr},
		aes256CTR:    {32, 16, AesCtr},
		aes128CFB:    {16, 16, AesCfb},
		aes192CFB:    {24, 16, AesCfb},
		aes256CFB:    {32, 16, AesCfb},
		salsa20_:     {32, 8, Salsa20},
		bfCFB:        {16, 8, BfCfb},
		rc4Md5:       {16, 16, Rc4Md5},
		chacha20_:    {32, 8, Chacha20},
		chacha20IETF: {32, 12, Chacha20Ietf},
	}

	aeadCipherMap = map[string]aeadCipherWrapper{
		aes128GCM:         {16, 16, 12, 16, AesGcm},
		aes192GCM:         {24, 24, 12, 16, AesGcm},
		aes256GCM:         {32, 32, 12, 16, AesGcm},
		chacha20Poly1305:  {32, 32, 12, 16, Chacha20Poly1305},
		xchacha20Poly1305: {32, 32, 12, 16, XChacha20Poly1305},
	}
)

type (
	streamCipherWrapper struct {
		KeySize   int
		IVSize    int
		NewCipher func(key []byte, ivSize int) (StreamCipher, error)
	}

	aeadCipherWrapper struct {
		KeySize   int
		SaltSize  int
		NonceSize int
		TagSize   int
		NewCipher func(key []byte, saltSize int) (AEADCipher, error)
	}
)

func GetCipher(method, password string) (sc StreamCipher, ac AEADCipher, err error) {
	if method == "none" {
		return nil, nil, nil
	}
	if _, ok := streamCipherMap[method]; ok {
		sc, err = NewStreamCipher(method, password)
	} else if _, ok = aeadCipherMap[method]; ok {
		ac, err = NewAEADCipher(method, password)
	} else {
		err = fmt.Errorf("unsupported method: %s", method)
	}
	return
}

func NewStreamCipher(method, password string) (StreamCipher, error) {
	x, ok := streamCipherMap[method]
	if !ok {
		return nil, fmt.Errorf("unsupported stream cipher: %s", method)
	}
	// simple EVP_BytesToKey()
	key := Kdf(password, x.KeySize)
	return x.NewCipher(key, x.IVSize)
}

func NewAEADCipher(method, password string) (AEADCipher, error) {
	x, ok := aeadCipherMap[method]
	if !ok {
		return nil, fmt.Errorf("unsupported aead cipher: %s", method)
	}
	// simple EVP_BytesToKey()
	key := Kdf(password, x.KeySize)
	return x.NewCipher(key, x.SaltSize)
}

/*
#include <openssl/evp.h>
int EVP_BytesToKey(

	const EVP_CIPHER *type,
	const EVP_MD *md,
	const unsigned char *salt,
	const unsigned char *data,
	int datal,
	int count,
	unsigned char *key,
	unsigned char *iv);
*/
func Kdf(password string, keyLen int) []byte {
	var res, prev []byte
	h := md5.New()
	for len(res) < keyLen {
		h.Write(prev)
		h.Write([]byte(password))
		res = h.Sum(res)
		prev = res[len(res)-h.Size():]
		h.Reset()
	}
	return res[:keyLen]
}

func hkdfSha1(key, salt, info, outKey []byte) {
	r := hkdf.New(sha1.New, key, salt, info)
	if _, err := io.ReadFull(r, outKey); err != nil {
		logger.Logger.ErrorBy(err)
	}
}
