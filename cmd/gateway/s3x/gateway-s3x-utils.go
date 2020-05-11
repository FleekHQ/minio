package s3x

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"

	"fmt"
	minio "github.com/minio/minio/cmd"
	// logHttp "github.com/minio/minio/logger/target/http"
	"log"
	"net/http"
	"encoding/json"

	"strconv"
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

// API is a property on handlerinput.entry
type API struct {
	Bucket string `json:bucket` // bucket slug
	Name string		`json:name` 	//operation
	Object string	`json:object,omitempty`	// optional object key
}

// Entry is a property for handler input
type Entry struct {
	API API														`json:api`
	RequestHeader map[string]string		`json:requestHeader,omitempty`
	ResponseHeader map[string]string	`json:responseHeader,omitempty`
}

// HandlerInput is a custom input object for calling handler
type HandlerInput struct {
	Entry Entry	`json:entry`
	Hash string `json:hash`
}

// LambdaResponseHeaders is subset of http headers for our lambda request
type LambdaResponseHeaders struct {
	ContentType string `json:"Content-Type"`
}

// LambdaResponse to store response from lambda
type LambdaResponse struct {
	StatusCode int                    `json:"statusCode"`
	Headers    LambdaResponseHeaders 	`json:"headers"`
}

func callPutBucketHandler(userID string, bucket string, hash string) error {
	requestHeader := make(map[string]string) 
	requestHeader["Authorization"] = userID
	api := API{
		Bucket: bucket,
		Name: "PutBucket",
	}
	entry:= Entry{
		API: api,
		RequestHeader: requestHeader,
	}
	handlerInput := &HandlerInput{
		Entry:entry,
		Hash: hash,
	}
	j, err:= json.Marshal(handlerInput)
	if err != nil {
		fmt.Println("error marshaling json: ", err)
	}
	// QUESTION: why is this printing field names in caps?
	log.Println("TODO: call lambda with \r\n", string(j))

	// Time to call lambda
	// https://github.com/awsdocs/aws-doc-sdk-examples/blob/master/go/example_code/lambda/aws-go-sdk-lambda-example-run-function.go
	
	// Create Lambda service client
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	client := lambda.New(sess, &aws.Config{Region: aws.String("us-west-2")})

	// TODO: env var for stage
	result, err := client.Invoke(&lambda.InvokeInput{FunctionName: aws.String("s3x-dev-crudHandler"), Payload: j})
	if err != nil {
			fmt.Println("Error calling create bucket handler")
			return fmt.Errorf("Error calling create bucket handler")
	}

	var resp LambdaResponse

	err = json.Unmarshal(result.Payload, &resp)
	if err != nil {
			fmt.Println("Error unmarshalling create bucket handler response")
			return fmt.Errorf("Error unmarshalling create bucket handler response")
	}

	// If the status code is NOT 200, the call failed
	if resp.StatusCode != 200 {
			fmt.Println("Error calling create bucket handler, StatusCode: " + strconv.Itoa(resp.StatusCode))
			return fmt.Errorf("Error calling create bucket handler, StatusCode: " + strconv.Itoa(resp.StatusCode))
	}

	return nil
}