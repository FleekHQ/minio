package s3x

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
)



const (
	authHeader = "Authorization"
)

type OperationHelper interface {
	CallPutBucketHandler(ctx context.Context, bucket string, hash string) error
	CallPutObjectHandler(ctx context.Context, bucket string, hash string, object string) error
}

type Invoker interface {
	Invoke(input *lambda.InvokeInput) (*lambda.InvokeOutput, error)
}

type lambdaHelper struct {
	client Invoker
}

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

func NewOperationHelper(i Invoker) OperationHelper {
	client := i
	if client == nil {
		sess := session.Must(session.NewSession())

		client = lambda.New(sess, &aws.Config{
			Region: aws.String("us-west-2"),
		})
	}
	oh := &lambdaHelper{
		client: client,
	}

	return oh
}

func (lh *lambdaHelper) CallPutBucketHandler(ctx context.Context, bucket string, hash string) error {
	return lh.callHandlerOperation(ctx, bucket, hash, "PutBucket", "")
}

func (lh *lambdaHelper) CallPutObjectHandler(ctx context.Context, bucket string, hash string, object string) error {
	return lh.callHandlerOperation(ctx, bucket, hash, "PutObject", object)
}

func (lh *lambdaHelper) callHandlerOperation(ctx context.Context, bucket string, hash string, operation string, object string) error {
	requestHeader := make(map[string]string)
	responseHeader := make(map[string]string)
	authHeader, err := extractAuthHeader(ctx)
	if err != nil {
		return ErrLambdaHandler
	}

	requestHeader["Authorization"] = authHeader
	responseHeader["X-FLEEK-IPFS-HASH"] = hash
	api := API{
		Bucket: bucket,
		Name:   operation,
		Object: object,
	}
	entry := Entry{
		API:            api,
		RequestHeader:  requestHeader,
		ResponseHeader: responseHeader,
	}
	handlerInput := &HandlerInput{
		Entry: entry,
		Hash:  hash,
	}
	j, err := json.Marshal(handlerInput)
	if err != nil {
		log.Println("error marshaling json: ", err)
		return ErrLambdaHandler
	}

	log.Println("calling lambda with: ", string(j))

	// Time to call lambda
	// https://github.com/awsdocs/aws-doc-sdk-examples/blob/master/go/example_code/lambda/aws-go-sdk-lambda-example-run-function.go

	if lh.client == nil {
		sess := session.Must(session.NewSession())

		lh.client = lambda.New(sess, &aws.Config{
			Region: aws.String("us-west-2"),
		})
	}


	handlerFunction := os.Getenv("CRUD_HANDLER_FUNCTION")

	result, err := lh.client.Invoke(&lambda.InvokeInput{
		FunctionName: aws.String(handlerFunction),
		Payload:      j})
	if err != nil {
		log.Println("got error: ", err)
		log.Println(fmt.Sprintf("Error calling create bucket handler: %s", err))
		return ErrLambdaHandler
	}

	log.Println(fmt.Sprintf("result: %v", result))

	var resp LambdaResponse

	// TODO: add statusCode to lambda response
/*	err = json.Unmarshal(result.Payload, &resp)
	if err != nil {
		log.Println("Error unmarshalling create bucket handler response")
		return ErrLambdaHandler
	}
*/
	// If the status code is NOT 200, the call failed
	if *result.StatusCode != 200 {
		log.Println("Error calling create bucket handler, StatusCode: " + strconv.Itoa(resp.StatusCode))
		return ErrLambdaHandler
	}

	return nil
}

func extractAuthHeader(ctx context.Context) (string, error) {
	var auth string
	var ok bool
	headerErrMsg := "error extracting auth header from context"
	if auth, ok = ctx.Value(authHeader).(string); !ok {
		log.Println(headerErrMsg)
		return "", ErrLambdaHandler
	}
	if auth == "" {
		log.Println(headerErrMsg)
		return "", ErrLambdaHandler
	}
	return auth, nil
}
