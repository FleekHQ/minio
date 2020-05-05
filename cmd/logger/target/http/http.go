/*
 * MinIO Cloud Storage, (C) 2018, 2019 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	xhttp "github.com/minio/minio/cmd/http"
	"github.com/minio/minio/cmd/logger/message/audit"
)

const (
	IpfsHashHeader      = "CONTENT-IPFS-HASH"
	IpfsHashContentType = "IPFS-CONTENT-TYPE"
	InfoEndpoint        = "http://localhost:8889/info?"
)

var (
	supportedApiEvents = []string{
		"PutBucket",
		"DeleteBucket",
		"DeleteMultipleObjects",
		"PutObject",
		"CopyObject",
		"DeleteObject",
		"NewMultipartUpload",
		// not sure about these below
		"CompleteMultipartUpload",
		"AbortMultipartUpload",
		"PutObjectPart",
		"CopyObjectPart",
	}
)

// Target implements logger.Target and sends the json
// format of a log entry to the configured http endpoint.
// An internal buffer of logs is maintained but when the
// buffer is full, new logs are just ignored and an error
// is returned to the caller.
type Target struct {
	// Channel of log entries
	logCh chan interface{}

	// HTTP(s) endpoint
	endpoint string
	// Authorization token for `endpoint`
	authToken string
	// User-Agent to be set on each log to `endpoint`
	userAgent string
	logKind   string
	client    http.Client
}

// FLEEK ADDED LOGGING TYPES ****

type ipfsInfoResponse struct {
	Bucket string `json:"bucket,omitempty"`
	Object string `json:"object,omitempty"`
	Hash   string `json:"hash,omitempty"`
}

type hashResponseHelper struct {
	Hash            string
	IpfsContentType string
}

type logFormattedEntry struct {
	Entry           interface{} `json:"entry,omitempty"`
	IpfsHash        string      `json:"hash,omitempty"`
	IpfsContentType string      `json:"contentType,omitempty"`
}

func (h *Target) startHTTPLogger() {
	// Create a routine which sends json logs received
	// from an internal channel.
	go func() {
		for entry := range h.logCh {
			logJSON, err := json.Marshal(&entry)
			if err != nil {
				continue
			}

			req, err := http.NewRequest(http.MethodPost, h.endpoint, bytes.NewReader(logJSON))
			if err != nil {
				continue
			}
			req.Header.Set(xhttp.ContentType, "application/json")

			// Set user-agent to indicate MinIO release
			// version to the configured log endpoint
			req.Header.Set("User-Agent", h.userAgent)

			if h.authToken != "" {
				req.Header.Set("Authorization", h.authToken)
			}

			// add ipfshash to headers
			infoRes, err := h.getHashFromEntry(entry)
			if err == nil {
				req.Header.Set(IpfsHashHeader, infoRes.Hash)
				req.Header.Set(IpfsHashContentType, infoRes.IpfsContentType)
				if  err := logEntry(entry, infoRes.Hash, infoRes.IpfsContentType); err != nil {
					log.Println("unable to log entry for fleek event: " + string(logJSON))
				}
			}

			resp, err := h.client.Do(req)
			if err != nil {
				h.client.CloseIdleConnections()
				continue
			}

			// Drain any response.
			xhttp.DrainBody(resp.Body)
		}
	}()
}

// Option is a function type that accepts a pointer Target
type Option func(*Target)

// WithEndpoint adds a new endpoint
func WithEndpoint(endpoint string) Option {
	return func(t *Target) {
		t.endpoint = endpoint
	}
}

// WithLogKind adds a log type for this target
func WithLogKind(logKind string) Option {
	return func(t *Target) {
		t.logKind = strings.ToUpper(logKind)
	}
}

// WithUserAgent adds a custom user-agent sent to the target.
func WithUserAgent(userAgent string) Option {
	return func(t *Target) {
		t.userAgent = userAgent
	}
}

// WithAuthToken adds a new authorization header to be sent to target.
func WithAuthToken(authToken string) Option {
	return func(t *Target) {
		t.authToken = authToken
	}
}

// WithTransport adds a custom transport with custom timeouts and tuning.
func WithTransport(transport *http.Transport) Option {
	return func(t *Target) {
		t.client = http.Client{
			Transport: transport,
		}
	}
}

// New initializes a new logger target which
// sends log over http to the specified endpoint
func New(opts ...Option) *Target {
	h := &Target{
		logCh: make(chan interface{}, 10000),
	}

	// Loop through each option
	for _, opt := range opts {
		// Call the option giving the instantiated
		// *Target as the argument
		opt(h)
	}

	h.startHTTPLogger()
	return h
}

// Send log message 'e' to http target.
func (h *Target) Send(entry interface{}, errKind string) error {
	/*	if h.logKind != errKind && h.logKind != "ALL" {
		return nil
	}*/
	var ok bool
	var entryStruct *audit.Entry
	if entryStruct, ok = checkEntry(entry); !ok {
		return nil
	}

	select {
	case h.logCh <- entryStruct:
	default:
		// log channel is full, do not wait and return
		// an error immediately to the caller
		return errors.New("log buffer full")
	}

	return nil
}


/****** NOTE: @dougmolina ADDED LOGIC TO SUPPORT FLEEK EVENTS *****/
// TODO: maybe make both audit Entry a non pointer to keep it cohesive across all funcs
func (h *Target) getHashFromEntry(entry interface{}) (*hashResponseHelper, error) {
	var ok bool
	var entryStruct *audit.Entry
	if entryStruct, ok = entry.(*audit.Entry); !ok {
		logCastMsg := "system Error trying to cast audit log entry"
		log.Println(logCastMsg)
		return nil, errors.New(logCastMsg)
	}

	// build endpoint URL
	bucketName := entryStruct.API.Bucket
	var endpoint string
	endpoint = InfoEndpoint + "bucket=" + bucketName

	if entryStruct.API.Object != "" {
		endpoint = endpoint + "&object=" + entryStruct.API.Object
	}

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		log.Println("Error building request for objectInfo :" + err.Error())
		return nil, err
	}

	resp, err := h.client.Do(req)
	if err != nil {
		log.Println("Error fetching response for objectInfo :" + err.Error())
		h.client.CloseIdleConnections()
		return nil, err
	}

	bytes, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Println("Error parsing body response for objectInfo :" + err.Error())
		return nil, err
	}

	var infoResponse ipfsInfoResponse
	if err := json.Unmarshal(bytes, &infoResponse); err != nil {
		log.Println("Error parsing json body response for objectInfo :" + err.Error())
		return nil, err
	}
	// Drain any response.
	defer xhttp.DrainBody(resp.Body)

	var ipfsContentType string
	if infoResponse.Object == "" {
		ipfsContentType = "Bucket"
	} else {
		ipfsContentType = "Object"
	}

	return &hashResponseHelper{
		Hash:            infoResponse.Hash,
		IpfsContentType: ipfsContentType,
	}, nil
}

func contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

// checks for Events that we want to send
func checkEntry(entry interface{}) (*audit.Entry, bool) {
	var ok bool
	var entryStruct audit.Entry
	if entryStruct, ok = entry.(audit.Entry); !ok {
		log.Println("system Error trying to cast audit log entry")
		return nil, false
	}

	if entryStruct.API.StatusCode == 0 {
		log.Println("system Error unable to read status code to determine logging payload")
		return nil, false
	}

	if entryStruct.API.StatusCode > 299 {
		return nil, false
	}

	if contains(supportedApiEvents, entryStruct.API.Name) {
		return &entryStruct, true
	}

	return nil, false
}

func logEntry(entry interface{}, hash string, contentType string) error {
	logEntry := logFormattedEntry{
		Entry:           entry,
		IpfsHash:        hash,
		IpfsContentType: contentType,
	}
	logJSON, err := json.Marshal(&logEntry)
	if err != nil {
		return errors.New("unable to marshal log entry")
	}

	log.Println("FLEEK CRUD: " + string(logJSON))

	return nil
}