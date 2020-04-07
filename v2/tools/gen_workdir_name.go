package main

import (
	"fmt"
	"hash/adler32"
	"log"
)

func main() {
	alphabet := []byte("abcdefghijklmnopqrstuvwxyz")
	fmt.Printf("Alphabet: %+s\n", alphabet)

	samplePath := "/root/ddgs_x64"

	adlHdl := adler32.New()
	_, err := adlHdl.Write([]byte(samplePath))
	if err != nil {
		log.Fatal(err)
	}
	s1 := adlHdl.Sum(nil)
	fmt.Printf("R 1 hash: %x\n", s1)

	_, err = adlHdl.Write(s1)
	if err != nil {
		log.Fatal(err)
	}
	s2 := adlHdl.Sum(nil)
	fmt.Printf("R 2 hash: %x\n", s2)

	var workDirNameBytes []byte
	for _, b := range s2 {
		idx := b % 26
		workDirNameBytes = append(workDirNameBytes, alphabet[idx])
	}

	fmt.Println("Work directory name: ", string(workDirNameBytes))
}
