package cipher

import (
	"crypto/aes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"io"
	"testing"

	"github.com/josexy/mini-ss/address"
)

func Test_hkdfSha1(t *testing.T) {
	key := []byte("helloworld")
	slat := []byte("salt")
	info := []byte("info")
	outkey := make([]byte, 16)
	outkey2 := make([]byte, 16)
	hkdfSha1(key, slat, info, outkey)
	t.Log(hex.EncodeToString(outkey))
	t.Log(hex.EncodeToString(outkey2))
}

func Test_Kdf(t *testing.T) {
	password := string("this is a password")
	t.Log(hex.EncodeToString(Kdf(password, 16)))
}

func testEncDec(t *testing.T, keySize, ivSize int, f func([]byte, int) (StreamCipher, error)) {

	// key := bytes.Repeat([]byte("1"), keySize)
	// iv := bytes.Repeat([]byte("1"), ivSize)

	key := make([]byte, keySize)
	iv := make([]byte, ivSize)
	io.ReadFull(rand.Reader, key)
	io.ReadFull(rand.Reader, iv)

	sc, err := f(key, ivSize)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("key: %s, len: %d", hex.EncodeToString(key), len(key))
	t.Logf("iv: %s, len: %d", hex.EncodeToString(iv), len(iv))

	// encrypt
	encCipher, err := sc.Encrypter(iv)
	if err != nil {
		t.Fatal(err)
	}
	src := []byte("hello world")
	dest := make([]byte, len(src))

	encCipher.XORKeyStream(dest, src)
	t.Log("hex:", hex.EncodeToString(dest), len(dest))
	t.Log("base64:", base64.StdEncoding.EncodeToString(dest))

	// decrypt
	decCipher, err := sc.Decrypter(iv)
	if err != nil {
		t.Fatal(err)
	}
	src2 := make([]byte, len(dest))
	decCipher.XORKeyStream(src2, dest)
	t.Log(string(src2), len(src2))
}

func Test_AESCTR(t *testing.T) {
	testEncDec(t, aes.BlockSize, aes.BlockSize, AESCTR)
}

func Test_AESCFB(t *testing.T) {
	testEncDec(t, aes.BlockSize, aes.BlockSize, AESCFB)
}

func TestAESEncDec(t *testing.T) {
	key, _ := hex.DecodeString("827ccb0eea8a706c4c34a16891f84e7b")
	iv, _ := hex.DecodeString("1e4ab798b9139c84a3ce7ba440f3cb9a")
	sc, _ := AESCFB(key, len(iv))
	s, _ := sc.Decrypter(iv)

	src, _ := hex.DecodeString("48dcd76c856769f77b72e22305bb479d7bd5ea3132a9540d6aac32f511f66d833e47b8db54c49ed54386bb6ca00fcfc049c6e497810948bf8390bcf3503b4f732873c05bbc447ca9ad09d30799c1498c4a3371cb")
	dest := make([]byte, len(src))

	s.XORKeyStream(dest, src)
	addr := address.ParseAddress3(dest)
	t.Log(addr)
	t.Log(dest[len(addr):])
	t.Log(string(dest[len(addr):]))
}

func TestChacah20(t *testing.T) {
	testEncDec(t, 32, 12, CHACHA20)
}

func TestBfCFB(t *testing.T) {
	testEncDec(t, 16, 8, BFCFB)
}

func TestAESGCM(t *testing.T) {
	// key := bytes.Repeat([]byte("1"), keySize)
	// iv := bytes.Repeat([]byte("1"), ivSize)

	key := make([]byte, 16)
	salt := make([]byte, 16)
	io.ReadFull(rand.Reader, key)
	io.ReadFull(rand.Reader, salt)

	aead, err := AESGCM(key, 16)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("key: %s, len: %d", hex.EncodeToString(key), len(key))
	t.Logf("salt: %s, len: %d", hex.EncodeToString(salt), len(salt))

	// encrypt
	encCipher, err := aead.Encrypter(salt)
	if err != nil {
		t.Fatal(err)
	}

	nonce := make([]byte, encCipher.NonceSize())
	t.Log("nonce: ", hex.EncodeToString(nonce))

	src, _ := hex.DecodeString("aabbccddeeff")
	buf := make([]byte, len(src)+encCipher.Overhead())

	dest := encCipher.Seal(buf[:0], nonce, src, nil)
	t.Log("hex:", hex.EncodeToString(dest), len(dest))
	t.Log("hex:", hex.EncodeToString(buf), len(buf))
	t.Log("base64:", base64.StdEncoding.EncodeToString(dest))

	// decrypt
	decCipher, err := aead.Decrypter(salt)
	if err != nil {
		t.Fatal(err)
	}
	src2, err := decCipher.Open(nil, nonce, dest, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(hex.EncodeToString(src2), len(src2))
}
