package s3x

import (
	"fmt"
	minio "github.com/minio/minio/cmd"
	"log"
	"net/http"
)

/* Design Notes
---------------

These functions should never call `toMinioErr`, and instead bubble up the errors.
Any error parsing to return minio errors should be done in the calling S3 functions.
*/

// getMinioObjectInfo is used to convert between object info in our protocol buffer format, to a minio object layer info type
func getMinioObjectInfo(o *ObjectInfo) minio.ObjectInfo {
	if o == nil {
		return minio.ObjectInfo{}
	}
	return minio.ObjectInfo{
		Bucket:      o.Bucket,
		Name:        o.Name,
		ETag:        minio.ToS3ETag(o.Etag),
		Size:        o.Size_,
		ModTime:     o.ModTime,
		ContentType: o.ContentType,
		UserDefined: o.UserDefined,
	}
}

// ******  FLEEK UTILS *************

func pingHash(hash string) {
	// PING hashes to Temporal and FLEEK gateway
	go func () {
		_, err := http.Get("https://gateway.temporal.cloud/ipfs/" + hash)
		if err != nil {
			log.Println(fmt.Printf("error when pinging temporal gateway on hash %s. Err: %s", hash, err.Error()))
		}
		log.Println("pinged to temporal gateway on hash " + hash)
	} ()

	go func () {
		_, err := http.Get("https://ipfs.fleek.co/ipfs/" + hash)
		if err != nil {
			log.Println(fmt.Printf("error when pinging fleek gateway on hash %s. Err: %s", hash, err.Error()))
		}
		log.Println("pinged to Fleek gateway on hash " + hash)
	} ()
}