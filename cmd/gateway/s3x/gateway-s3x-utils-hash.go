package s3x

import (
	"fmt"
	"github.com/ipfs/go-cid"
	"log"
	"net/http"
)

// ******  FLEEK UTILS *************

func pingHash(hash string) {
	// PING hashes on IPFS gateways
	urls := []string{
		"https://gateway.temporal.cloud/ipfs/" + hash,
		"https://ipfs.fleek.co/ipfs/" + hash,
		"https://ipfs.io/ipfs/" + hash,
	}
	for _, url := range urls {
		go func (url string) {
			_, err := http.Get(url)
			if err != nil {
				log.Println(fmt.Printf("error when pinging url %s on hash %s. Err: %s", url, hash, err.Error()))
			}
			log.Println(fmt.Sprintf("pinged to url gateway %s with hash %s", url, hash))
		} (url)
	}
}

func convertToHashV0(hash string) (string) {
	c, err := cid.Decode(hash)
	if err != nil {
		log.Println("error trying to convert hash to V0", hash)
		return ""
	}

	if c.Version() != 0 {
		// cid if not V0
		return c.Hash().B58String()
	}

	return hash
}