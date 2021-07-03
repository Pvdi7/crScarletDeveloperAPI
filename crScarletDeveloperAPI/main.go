package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	b64 "encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GenerateRandomString(n int) (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		ret[i] = letters[num.Int64()]
	}

	return string(ret), nil
}

func developerAPI(c *gin.Context) {

    certID := c.PostForm("certID")
    publicKey := c.PostForm("publicKey")

    /*
        What do we do ?
            1. Generate key and encrypt certData
            2. Concatinate information
            3. Send encrypted cert and encrypted key as json
    
    */

        /* key must be 32 bytes AES256 */
        stringKey, _ := GenerateRandomString(32)
        var key = []byte(stringKey)
        
        // Create Cipher Block...
        block, err := aes.NewCipher(key)
        if err != nil {
            fmt.Println(err)
        }
        
        aesGCM, err := cipher.NewGCM(block)
        if err != nil {
            fmt.Println(err)
        }

        // Create GCM nonce
        nonce := make([]byte, aesGCM.NonceSize())
        if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
            fmt.Println(err)
        }
        
        certData, err := ioutil.ReadFile("certs/" + certID + "/cert.dcrscarlet")
        
        if err != nil {
            // Handle error...
            fmt.Println(err)
        }

    // Encrypt data
    encryptedData := aesGCM.Seal(nil, nonce, certData, nil)
    
    /*
        Encrypt key using rsa....
    */

    keyBlock, _ := pem.Decode([]byte(publicKey))
    if block == nil {
        fmt.Println("Err: Failed to parse PEM block....")
    }

    pub, err := x509.ParsePKCS1PublicKey(keyBlock.Bytes)
    if err != nil {
        fmt.Println("Err: Failed to parse encoded public key...")
    }

    encryptedKey, err := rsa.EncryptPKCS1v15(
        rand.Reader,
        pub,
        key,
    )

    /* 
        GCM nonce is 12 bytes and GCM tag is 16 bytes in size...
            - call AEAD.Overhead() for length of the tag
            - call AEAD.NonceSize() for length of the nonce

        By default this framework appends the tag to the end of the cipher

        Scarlet expects first 12 bytes to be the nonce, middle to be the certificate, and last 16 bytes to be the tag....
        Hoping that this notice can help others who want to use something other than golang
    */
    
    encryptedData = append(nonce, encryptedData...)

    /* Send as json the concatinated data and encryptedKey.... */
    c.JSON(http.StatusOK, gin.H{
        "key": encryptedKey,
        "cert": b64.StdEncoding.EncodeToString(encryptedData),
    })

}

func main() {
    r := gin.Default()

    r.POST("/developerAPI", developerAPI)

    r.Run(":4000")
}
