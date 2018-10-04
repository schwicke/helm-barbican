package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

// encryptCmd represents the hello command
var encryptCmd = &cobra.Command{
	Use:   "enc",
	Short: "encrypt deployment secrets",
	Long:  `This command `,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		content, err := ioutil.ReadFile(SecretsFile)
		if err != nil {
			log.Fatalf("encrypt failed : %v", err)
		}
		if b64Encoded(string(content)) {
			log.Fatal("content is empty or already encrypted")
		}

		client, err := newKeyManager()
		if err != nil {
			log.Fatalf("could not init client :: %v", err)
		}
		key, nonce, err := fetchKey(client, Deployment)
		if err != nil {
			log.Fatalf("could not fetch key : %v", err)
		}

		result, err := encrypt(key, nonce, content)
		err = ioutil.WriteFile(SecretsFile, result, 0644)
		if err != nil {
			log.Fatalf("encrypt failed : %v", err)
		}

	},
}

// decryptCmd represents the hello command
var decryptCmd = &cobra.Command{
	Use:   "dec",
	Short: "decrypt deployment secrets",
	Long:  `This command `,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		content, err := ioutil.ReadFile(SecretsFile)
		if err != nil {
			log.Fatalf("decrypt failed : %v", err)
		}
		if !b64Encoded(string(content)) {
			log.Fatal("not touching unencrypted content")
		}
		client, err := newKeyManager()
		if err != nil {
			log.Fatalf("could not init client :: %v", err)
		}
		key, nonce, err := fetchKey(client, Deployment)
		if err != nil {
			log.Fatalf("could not get key : %v", err)
		}
		plain, err := decrypt(key, nonce, string(content))
		if err != nil {
			log.Fatalf("decrypt failed : %v", err)
		}
		err = ioutil.WriteFile(SecretsFile, plain, 0644)
		if err != nil {
			log.Fatalf("could not write file : %v", err)
		}
	},
}

// viewCmd
var viewCmd = &cobra.Command{
	Use:   "view",
	Short: "view deployment secrets",
	Long:  `This command `,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		content, err := ioutil.ReadFile(SecretsFile)
		if err != nil {
			log.Fatalf("decrypt failed : %v", err)
		}
		if b64Encoded(string(content)) {
			client, err := newKeyManager()
			if err != nil {
				log.Fatalf("could not init client :: %v", err)
			}
			key, nonce, err := fetchKey(client, Deployment)
			if err != nil {
				log.Fatalf("could not get key :: %v", err)
			}
			content, err = decrypt(key, nonce, string(content))
			if err != nil {
				log.Fatalf("decrypt failed : %v", err)
			}
		}
		fmt.Printf("%v", string(content))
	},
}

// editCmd
var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "edit deployment secrets",
	Long:  `This command`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := newKeyManager()
		if err != nil {
			log.Fatalf("could not init client :: %v", err)
		}
		key, nonce, err := fetchKey(client, Deployment)
		if err != nil {
			log.Fatalf("could not fetch key : %v", err)
		}
		content, err := ioutil.ReadFile(SecretsFile)
		if err != nil {
			log.Fatalf("decrypt failed : %v", err)
		}
		if b64Encoded(string(content)) {
			content, err = decrypt(key, nonce, string(content))
			if err != nil {
				log.Fatalf("decrypt failed : %v", err)
			}
		}
		ed := NewEditor()
		result, _, err := ed.LaunchTemp("ppp", "sss", strings.NewReader(string(content)))
		encrypted, err := encrypt(key, nonce, result)
		err = ioutil.WriteFile(SecretsFile, encrypted, 0644)
		if err != nil {
			log.Fatalf("failed to encrypt : %v", err)
		}
	},
}

func newKey() (string, string, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", "", err
	}

	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", err
	}

	return base64.StdEncoding.EncodeToString(key), base64.StdEncoding.EncodeToString(nonce), nil
}

func encrypt(b64key string, b64nonce string, payload []byte) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(b64key)
	if err != nil {
		return nil, err
	}
	nonce, err := base64.StdEncoding.DecodeString(b64nonce)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	sealed := aesgcm.Seal(nil, nonce, payload, nil)
	result := make([]byte, base64.StdEncoding.EncodedLen(len(sealed)))
	base64.StdEncoding.Encode(result, sealed)
	return result, nil
}

func decrypt(b64key string, b64nonce string, b64payload string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(b64key)
	if err != nil {
		return nil, err
	}
	nonce, err := base64.StdEncoding.DecodeString(b64nonce)
	if err != nil {
		return nil, err
	}
	payload, err := base64.StdEncoding.DecodeString(b64payload)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := aesgcm.Open(nil, nonce, payload, nil)
	if err != nil {
		return nil, err
	}
	return plain, nil
}

func b64Encoded(content string) bool {
	_, err := base64.StdEncoding.DecodeString(content)
	if err == nil {
		return true
	}
	return false
}

func init() {
	RootCmd.AddCommand(encryptCmd)
	RootCmd.AddCommand(decryptCmd)
	RootCmd.AddCommand(viewCmd)
	RootCmd.AddCommand(editCmd)
}