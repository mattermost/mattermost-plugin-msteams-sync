package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPad(t *testing.T) {
	assert := assert.New(t)
	resp := pad(make([]byte, 3))
	assert.Equal([]byte{0x0, 0x0, 0x0, 0xd, 0xd, 0xd, 0xd, 0xd, 0xd, 0xd, 0xd, 0xd, 0xd, 0xd, 0xd, 0xd}, resp)
}

func TestUnpad(t *testing.T) {
	assert := assert.New(t)
	resp, err := unpad(make([]byte, 1))
	assert.Nil(err)
	assert.Equal([]byte{0x0}, resp)
}

func TestEncrypt(t *testing.T) {
	for _, test := range []struct {
		Name          string
		Key           []byte
		ExpectedError string
	}{
		{
			Name:          "Encrypt: Invalid key",
			Key:           make([]byte, 1),
			ExpectedError: "could not create a cipher block",
		},
		{
			Name: "Encrypt: Valid",
			Key:  make([]byte, 16),
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			assert := assert.New(t)
			encryptedKey, err := encrypt(test.Key, "mockData")
			if test.ExpectedError != "" {
				assert.Contains(err.Error(), test.ExpectedError)
				assert.Equal("", encryptedKey)
			} else {
				assert.Nil(err)
				assert.NotEqual("", encryptedKey)
			}

			decryptedKey, err := decrypt(test.Key, encryptedKey)
			if test.ExpectedError != "" {
				assert.Contains(err.Error(), test.ExpectedError)
				assert.Equal("", decryptedKey)
			} else {
				assert.Nil(err)
				assert.Equal("mockData", decryptedKey)
			}
		})
	}
}
