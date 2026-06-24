package login

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"regexp"
)

const publicKeyPEM = `
-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDLgd2OAkcGVtoE3ThUREbio0Eg
Uc/prcajMKXvkCKFCWhJYJcLkcM2DKKcSeFpD/j6Boy538YXnR6VhcuUJOhH2x71
nzPjfdTcqMz7djHum0qSZA0AyCBDABUqCrfNgCiJ00Ra7GmRj+YCK1NJEuewlb40
JNrRuoEUXpabUzGB8QIDAQAB
-----END PUBLIC KEY-----
`

func getCsrf(cookie string) string {
	reg := regexp.MustCompile(`bili_jct=([0-9a-zA-Z]+);`)
	if result := reg.FindStringSubmatch(cookie); len(result) > 1 {
		return result[1]
	}
	return ""
}

func getCorrespondPath(ts int64) (string, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return "", fmt.Errorf("failed to decode public key PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("not an RSA public key")
	}
	msg := fmt.Appendf(nil, "refresh_%d", ts)
	encrypted, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPub, msg, nil)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(encrypted), nil
}
