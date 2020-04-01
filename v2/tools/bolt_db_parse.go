package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/boltdb/bolt"
	"github.com/vmihailenco/msgpack"
)

var edKeySeed = []byte{
	0x5C, 0x9E, 0xAE, 0xAE, 0x43, 0x26, 0xB7, 0xA2,
	0x52, 0xDC, 0x43, 0xF9, 0xBD, 0x3F, 0xD1, 0xA6,
	0xC8, 0xB0, 0x28, 0xE1, 0xDF, 0xA8, 0xB0, 0xF5,
	0xCF, 0x43, 0xE7, 0x82, 0xD1, 0x90, 0x11, 0x6B,
}

var dbfd *bolt.DB
var edPubKey ed25519.PublicKey

func init() {
	var ok bool
	edPrivKey := ed25519.NewKeyFromSeed(edKeySeed)
	fmt.Printf("ED25519 Private Key:\n%s\n\n", hex.Dump(edPrivKey))

	cryptoPubKey := edPrivKey.Public()
	edPubKey, ok = cryptoPubKey.(ed25519.PublicKey)
	if ok {
		fmt.Printf("ED25519 Public Key:\n%s\n\n", hex.Dump(edPubKey))
	} else {
		fmt.Printf("ED25519 public key:\n%+v\n\n", edPubKey)
		log.Fatal("Failed to convert ed25519 public key.")
	}
	fmt.Println("-------------------------------------------------")
}

func getVal(elemRaw []byte) ([]byte, error) {
	aesIV := elemRaw[:aes.BlockSize]
	//fmt.Printf("AES IV:\n%s\n", hex.Dump(aesIV))

	aesKey := elemRaw[aes.BlockSize : aes.BlockSize+0x20]
	//fmt.Printf("AES Key:\n%s\n", hex.Dump(aesKey))

	edSig := elemRaw[0x30:0x70]
	//fmt.Printf("ED25519 Sig:\n%s\n", hex.Dump(edSig))

	elemCntntData := elemRaw[0x70:]

	sha512Hash := sha512.New()
	sha512Hash.Write(elemCntntData)
	sha512Val := sha512Hash.Sum(nil)
	//fmt.Printf("SHA512 value of passwd data:\n%s\n", hex.Dump(sha512Val))

	//fmt.Printf("ED25519 Publick Key len: %d\n\n", len(edPubKey))
	if ed25519.Verify(edPubKey, sha512Val, edSig) {
		aesBlock, err := aes.NewCipher(aesKey)
		if err != nil {
			return nil, fmt.Errorf("Failed to initialize aes block.\n%s\n", err.Error())
		}
		aesStream := cipher.NewCTR(aesBlock, aesIV)
		plainData := make([]byte, len(elemCntntData))
		aesStream.XORKeyStream(plainData, elemCntntData)
		fmt.Printf("AES Decryption succeed.\n\n")
		return plainData, nil

	}
	return nil, fmt.Errorf("Failed to ed25519.Verify()")
}

func main() {
	dbfd, err := bolt.Open("../data/v5019_bolt.db", 0600, &bolt.Options{Timeout: 30 * time.Second})
	if err != nil {
		log.Fatal(err)
	}

	defer dbfd.Close()

	dbfd.View(func(tx *bolt.Tx) error {
		//fmt.Printf("%+v\n", v)
		b := tx.Bucket([]byte("xproxy"))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			//fmt.Printf("key=%s, value=%s\n", k, v)
			fmt.Printf("Processing [%s]......\n\n", k)

			// Assume bucket exists and has keys
			plainData, err := getVal(v)
			if err != nil {
				fmt.Printf("Failed to parse dbKey [%s]\n%s\n", k, err.Error())
				continue
			}

			switch string(k) {
			case "seeds":
				{
					var seedList []map[string]interface{}
					err = msgpack.Unmarshal(plainData, &seedList)
					if err != nil {
						fmt.Printf("Failed to msgpack.Unmarshal seeds data.\n%s\n", err.Error())
						break
					}

					fmt.Println("Plain Seeds:")
					for _, s := range seedList {
						fmt.Println(s)
					}
				}
			case "hubsig":
				{
					var hubSigInfo map[string]interface{}
					err = msgpack.Unmarshal(plainData, &hubSigInfo)
					if err != nil {
						fmt.Printf("Failed to msgpack.Unmarshal hubsig info.\n%s\n", err.Error())
						break
					}
					fmt.Println("hubsig info:")
					fmt.Printf("hubsig['v']: %d\n", hubSigInfo["v"])
					fmt.Printf("hubsig['a']: %+v\n", hubSigInfo["a"])

					if pu, ok := hubSigInfo["pu"].([]byte); ok {
						fmt.Printf("hubsig['pu']:\n%s", hex.Dump(pu))
					} else {
						fmt.Println("Failed to convert hubsig['pu']")
					}

					if sig, ok := hubSigInfo["s"].([]byte); ok {
						fmt.Printf("hubsig['s']:\n%s\n", hex.Dump(sig))
					} else {
						fmt.Println("Failed to convert hubsig['s']")
					}
				}
			case "port":
				{
					var port int
					err = msgpack.Unmarshal(plainData, &port)
					if err != nil {
						fmt.Printf("Failed to msgpack.Unmarshal port number.\n%s\n", err.Error())
						break
					}
					fmt.Printf("Local proxy server port: %d\n", port)
				}
			default:
				{
					dstFile := string(k) + "_boltdb_parsed.mp"
					dstfd, err := os.Create(dstFile)
					if err != nil {
						fmt.Printf("Failed to create dst file to write.\n%s\n", err.Error())
						break
					}
					defer dstfd.Close()
					dstfd.Write(plainData)
				}
			}
			fmt.Println("-------------------------------------------------")
		}
		return nil
	})

}
