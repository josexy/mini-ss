package cipher

import (
	"crypto/aes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"io"
	"testing"

	"github.com/josexy/mini-ss/address"
	"github.com/stretchr/testify/assert"
)

func TestHkdfSha1(t *testing.T) {
	key := []byte("helloworld")
	slat := []byte("salt")
	info := []byte("info")
	outkey := make([]byte, 16)
	hkdfSha1(key, slat, info, outkey)
	t.Log(hex.EncodeToString(outkey))
}

func TestKdf(t *testing.T) {
	password := string("this is a password")
	t.Log(hex.EncodeToString(Kdf(password, 16)))
}

func testStreamCipherEncDec(t *testing.T, keySize, ivSize int, f func([]byte, int) (StreamCipher, error)) {
	key := make([]byte, keySize)
	iv := make([]byte, ivSize)
	io.ReadFull(rand.Reader, key)
	io.ReadFull(rand.Reader, iv)

	sc, err := f(key, ivSize)
	assert.Nil(t, err)

	t.Logf("key: %s, len: %d", hex.EncodeToString(key), len(key))
	t.Logf("iv: %s, len: %d", hex.EncodeToString(iv), len(iv))

	// encrypt
	encCipher, err := sc.GetEncrypter(iv)
	assert.Nil(t, err)

	origin := []byte("hello world")
	dest := make([]byte, len(origin))

	encCipher.XORKeyStream(dest, origin)
	t.Log("hex:", hex.EncodeToString(dest), len(dest))
	t.Log("base64:", base64.StdEncoding.EncodeToString(dest))

	// decrypt
	decCipher, err := sc.GetDecrypter(iv)
	assert.Nil(t, err)

	origin2 := make([]byte, len(dest))
	decCipher.XORKeyStream(origin2, dest)
	t.Logf("origin: %q, origin2: %q", string(origin), string(origin2))
	assert.Equal(t, origin, origin2)
}

func TestAesCtr(t *testing.T) {
	testStreamCipherEncDec(t, aes.BlockSize, aes.BlockSize, AesCtr)
}

func TestAesCfb(t *testing.T) {
	testStreamCipherEncDec(t, aes.BlockSize, aes.BlockSize, AesCfb)
}

func TestChacha20(t *testing.T) {
	testStreamCipherEncDec(t, 32, 12, Chacha20)
}

func TestBfCfb(t *testing.T) {
	testStreamCipherEncDec(t, 16, 8, BfCfb)
}

func TestAESEncDec(t *testing.T) {
	key, _ := hex.DecodeString("827ccb0eea8a706c4c34a16891f84e7b")
	iv, _ := hex.DecodeString("1e4ab798b9139c84a3ce7ba440f3cb9a")
	sc, _ := AesCfb(key, len(iv))
	s, _ := sc.GetDecrypter(iv)

	src, _ := hex.DecodeString("48dcd76c856769f77b72e22305bb479d7bd5ea3132a9540d6aac32f511f66d833e47b8db54c49ed54386bb6ca00fcfc049c6e497810948bf8390bcf3503b4f732873c05bbc447ca9ad09d30799c1498c4a3371cb")
	dest := make([]byte, len(src))

	s.XORKeyStream(dest, src)
	addr, err := address.ParseAddressFromBuffer(dest)
	assert.Nil(t, err)

	t.Log(addr.String())
	t.Log(hex.Dump(dest[len(addr):]))
	t.Logf("%q", string(dest[len(addr):]))
}

func TestAesGcm(t *testing.T) {
	key := make([]byte, 16)
	salt := make([]byte, 16)
	io.ReadFull(rand.Reader, key)
	io.ReadFull(rand.Reader, salt)

	aead, err := AesGcm(key, 16)
	assert.Nil(t, err)

	t.Logf("key: %s, len: %d", hex.EncodeToString(key), len(key))
	t.Logf("salt: %s, len: %d", hex.EncodeToString(salt), len(salt))

	// encrypt
	encCipher, err := aead.GetEncrypter(salt)
	assert.Nil(t, err)

	nonce := make([]byte, encCipher.NonceSize())
	t.Logf("nonce: %s", hex.EncodeToString(nonce))

	src, _ := hex.DecodeString("aabbccddeeff")
	buf := make([]byte, len(src)+encCipher.Overhead())

	dest := encCipher.Seal(buf[:0], nonce, src, nil)
	t.Log("hex:", hex.EncodeToString(dest), len(dest))
	t.Log("hex:", hex.EncodeToString(buf), len(buf))
	t.Log("base64:", base64.StdEncoding.EncodeToString(dest))

	// decrypt
	decCipher, err := aead.GetDecrypter(salt)
	assert.Nil(t, err)

	src2, err := decCipher.Open(nil, nonce, dest, nil)
	assert.Nil(t, err)
	t.Log(hex.EncodeToString(src), len(src))
	t.Log(hex.EncodeToString(src2), len(src2))
	assert.Equal(t, src, src2)
}
