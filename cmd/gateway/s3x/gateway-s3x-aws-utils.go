package s3x

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"

	"fmt"
	// logHttp "github.com/minio/minio/logger/target/http"
	"encoding/json"
	"log"
	"os"
	"strconv"
)

// API is a property on handlerinput.entry
type API struct {
	Bucket string `json:"bucket"`           // bucket slug
	Name   string `json:"name"`             //operation
	Object string `json:"object",omitempty` // optional object key
}

// Entry is a property for handler input
type Entry struct {
	API            API               `json:"api"`
	RequestHeader  map[string]string `json:"requestHeader",omitempty`
	ResponseHeader map[string]string `json:"responseHeader",omitempty`
}

// HandlerInput is a custom input object for calling handler
type HandlerInput struct {
	Entry Entry  `json:"entry"`
	Hash  string `json:"hash"`
}

// LambdaResponseHeaders is subset of http headers for our lambda request
type LambdaResponseHeaders struct {
	ContentType string `json:"Content-Type"`
}

// LambdaResponse to store response from lambda
type LambdaResponse struct {
	StatusCode int                   `json:"statusCode"`
	Headers    LambdaResponseHeaders `json:"headers"`
}

func callPutBucketHandler(userID string, bucket string, hash string) error {
	requestHeader := make(map[string]string)
	requestHeader["Authorization"] = userID
	api := API{
		Bucket: bucket,
		Name:   "PutBucket",
	}
	entry := Entry{
		API:           api,
		RequestHeader: requestHeader,
	}
	handlerInput := &HandlerInput{
		Entry: entry,
		Hash:  hash,
	}
	j, err := json.Marshal(handlerInput)
	if err != nil {
		fmt.Println("error marshaling json: ", err)
	}
	// QUESTION: why is this printing field names in caps?
	log.Println("calling lambda with: ", string(j))

	// Time to call lambda
	// https://github.com/awsdocs/aws-doc-sdk-examples/blob/master/go/example_code/lambda/aws-go-sdk-lambda-example-run-function.go

	// TODO: check for these on boot
	log.Println("AWS_ACCESS_KEY_ID: ", os.Getenv("AWS_ACCESS_KEY_ID"))
	log.Println("AWS_SECRET_ACCESS_KEY: ", os.Getenv("AWS_SECRET_ACCESS_KEY"))

	// Create Lambda service client
	// sess := session.Must(session.NewSessionWithOptions(session.Options{
	// 	SharedConfigState: session.SharedConfigEnable,
	// }))
	sess := session.Must(session.NewSession())

	client := lambda.New(sess, &aws.Config{
		Region:      aws.String("us-west-2"),
		Credentials: credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), ""),
	})

	// TODO: env var for stage
	// TODO: make lambda function come from env var too
	// NOTE: temp using httpLogger instead of s3x handler for testing
	result, err := client.Invoke(&lambda.InvokeInput{
		FunctionName:   aws.String("httpLogger"),
		InvocationType: aws.String("RequestResponse"),
		LogType:        aws.String("Tail"),
		Payload:        j})
	log.Println("returned from invoke")
	if err != nil {
		log.Println("got error: ", err)
		fmt.Sprintf("Error calling create bucket handler: %s", err)
		return fmt.Errorf("Error calling create bucket handler")
	}

	log.Println(fmt.Sprintf("result: %v", result))

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
