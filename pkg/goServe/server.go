package goServe

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/iambpn/go-server/pkg/request"
	"github.com/iambpn/go-server/pkg/response"
)

type globalErrorHandler = func(req *request.Request, res *response.Response, err error)

type urlPath struct {
	path        string
	pathPattern string
	splitPath   []string
	paramsIdx   []int
}

type endpoint struct {
	method string
	path   urlPath
	cb     func(req *request.Request, res *response.Response) error
}

type server struct {
	endpoints        []endpoint
	errorHandler     globalErrorHandler
	requestQueueSize int
	workerGoroutine  int
}

/*
address: "127.0.0.1:8080" "<ip>:<port>"
*/
func (s *server) Listen(address string) error {
	listener, err := net.Listen("tcp", address)

	if err != nil {
		return err
	}

	defer listener.Close()

	reqQueue := make(chan net.Conn, s.requestQueueSize)

	for i := 0; i < s.workerGoroutine; i++ {
		// this go routine must only read data from 's' (server)
		go workerFunc(*s, reqQueue)
	}

	fmt.Println("Listening on address: ", address)

	for {
		// pause for loop until request is arrived
		conn, err := listener.Accept()

		if err != nil {
			return err
		}

		reqQueue <- conn
	}
}

func workerFunc(s server, ch chan net.Conn) {
	for {
		conn := <-ch

		processRequest(&s, conn)
	}
}

func processRequest(s *server, conn net.Conn) {
	defer conn.Close()

	// handle uncaught errors/panics
	defer func() {
		panickedErr := recover()

		if panickedErr == nil {
			return
		}

		if err, ok := panickedErr.(error); ok {
			// Error
			log.Printf("Panicked Error: %v\n", err)
		} else {
			log.Printf("Panicked: %v\n", panickedErr)
		}

		// create a response and send to user
		res := response.New(conn.Write)
		res.JSON(response.JSONType{
			"error":      fmt.Errorf("%v", panickedErr).Error(),
			"message":    "Internal Server Error",
			"isPanicked": true,
		})
		res.StatusCode(500)
		res.Send()
	}()

	buffer := make([]byte, 1024)
	requestBytes := []byte{}

	for {
		// set timeout for 1ms
		err := conn.SetReadDeadline(time.Now().Add(time.Millisecond))

		if err != nil {
			panic(err)
		}

		readBytes, err := conn.Read(buffer)

		if err != nil {
			if err == io.EOF {
				// when connection is terminated by client side
				break
			}

			if errors.Is(err, os.ErrDeadlineExceeded) {
				break
			}

			panic(err)
		}

		requestBytes = append(requestBytes, buffer[:readBytes]...)
	}

	if len(requestBytes) <= 0 {
		return
	}

	method, path, httpVersion, headers := parseHtmlRequest(requestBytes)

	ep, err := s.matchEndpoint(method, path)

	params, query := parseParamsAndQuery(path, ep.path.paramsIdx, ep.path.splitPath)

	req := request.New(method, path, httpVersion, headers, params, query)

	res := response.New(conn.Write)

	if err != nil {
		s.errorHandler(req, res, err)
		return
	}

	err = ep.cb(req, res)

	if err != nil {
		s.errorHandler(req, res, err)
		return
	}
}

func (s *server) matchEndpoint(method string, path string) (*endpoint, error) {
	err := verifyMethod(method)

	if err != nil {
		return nil, err
	}

	for _, ep := range s.endpoints {
		err = verifyMethod(ep.method)

		if err != nil {
			return nil, err
		}

		isMatched, err := regexp.MatchString(ep.path.pathPattern, path)

		if err != nil {
			return nil, err
		}

		if isMatched && ep.method == strings.ToLower(method) {
			return &ep, nil
		}
	}

	return nil, errors.New("Path not found")
}

func parseParamsAndQuery(pathWithQuery string, paramsIdx []int, originalSplitPath []string) (params map[string]string, query map[string]string) {
	params = make(map[string]string)
	query = make(map[string]string)

	splitPathWithQuery := strings.Split(pathWithQuery, "?")

	pathUrl := splitPathWithQuery[0]
	splitPathUrl := strings.Split(pathUrl, "/")
	for _, idx := range paramsIdx {
		params[originalSplitPath[idx]] = splitPathUrl[idx]
	}

	if len(splitPathWithQuery) > 1 {
		queryString := splitPathWithQuery[1]
		for _, keyVal := range strings.Split(queryString, "&") {
			splitKeyVal := strings.Split(keyVal, "=")

			if len(splitKeyVal) <= 1 {
				continue
			}

			query[splitKeyVal[0]] = splitKeyVal[1]
		}
	}

	return params, query
}

func parseHtmlRequest(requestBytes []byte) (method string, path string, httpVersion string, headers map[string]string) {
	headerLines := strings.Split(string(requestBytes), "\n")

	// parse 1st line and implement different methods
	metadata := strings.Split(strings.TrimSpace(headerLines[0]), " ")

	method = metadata[0]
	path = metadata[1]
	httpVersion = metadata[2]

	if len(headerLines) > 1 {
		headers = parseHeaders(headerLines[1:])
	}

	return method, path, httpVersion, headers
}

func parseHeaders(headers []string) map[string]string {
	headersMap := make(map[string]string)

	for _, val := range headers {
		if val = strings.TrimSpace(val); val != "" {
			keyVal := strings.Split(val, ": ")
			headersMap[keyVal[0]] = strings.Join(keyVal[1:], ": ")
		}
	}

	return headersMap
}

func verifyMethod(method string) error {
	switch strings.ToLower(method) {
	case "get":
	case "post":
	case "put":
	case "patch":
	case "delete":
	default:
		return errors.New("method not supported")
	}

	return nil
}

var match_all_regex = "[^/]*"

/*
method is the standard REST Method
path is the app endpoints
handler is the endpoint handler
*/
func (s *server) AddPath(method string, path string, handler func(req *request.Request, res *response.Response) error) {
	parsedPath := urlPath{
		path:        path,
		pathPattern: "",
		splitPath:   strings.Split(path, "/"),
		paramsIdx:   []int{},
	}

	splitPatternPath := []string{}

	for idx, sp := range parsedPath.splitPath {
		if sp == "*" {
			// anything
			splitPatternPath = append(splitPatternPath, match_all_regex)
		} else if sp != "" && sp[0] == ':' {
			//params
			splitPatternPath = append(splitPatternPath, match_all_regex)
			parsedPath.paramsIdx = append(parsedPath.paramsIdx, idx)
		} else {
			splitPatternPath = append(splitPatternPath, sp)
		}
	}

	parsedPath.pathPattern = strings.Join(splitPatternPath, "/")

	s.endpoints = append(s.endpoints, endpoint{strings.ToLower(method), parsedPath, handler})
}

type ServerConfig struct {
	ErrorHandler globalErrorHandler
	QueueSize    int
	Worker       int
}

func New(configs ...ServerConfig) *server {
	config := ServerConfig{
		ErrorHandler: func(req *request.Request, res *response.Response, err error) {
			log.Println("Default Error Handler: ", err)

			if err != nil {
				err = res.JSON(response.JSONType{
					"error": err.Error(),
				})
			} else {
				res.StatusCode(500)
				res.JSON(response.JSONType{
					"error": "Internal Server Error",
				})
			}

			res.Send()
		},
		QueueSize: 1024,
		Worker:    runtime.NumCPU() * 10,
	}

	if len(configs) > 0 {
		if configs[0].ErrorHandler != nil {
			config.ErrorHandler = configs[0].ErrorHandler
		}

		if configs[0].QueueSize != 0 {
			config.QueueSize = configs[0].QueueSize
		}

		if configs[0].Worker != 0 {
			config.Worker = configs[0].Worker
		}
	}

	return &server{
		errorHandler:     config.ErrorHandler,
		requestQueueSize: config.QueueSize,
		workerGoroutine:  config.Worker,
	}
}
