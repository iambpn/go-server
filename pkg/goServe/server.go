package goServe

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
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
	endpoints    []endpoint
	errorHandler globalErrorHandler
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

	fmt.Println("Listening on address: ", address)

	for {
		// pause for loop until request is arrived
		conn, err := listener.Accept()

		if err != nil {
			return err
		}

		// this go routine must oly read data from s (server)
		go s.processRequest(conn)
	}
}

func (s *server) processRequest(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	requestBytes := []byte{}

	for {
		// set timeout for 1ms
		err := conn.SetReadDeadline(time.Now().Add(time.Millisecond))

		if err != nil {
			log.Fatalln(err)
		}

		readBytes, err := conn.Read(buffer)

		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				break
			}

			log.Fatalln(err)
		}

		requestBytes = append(requestBytes, buffer[:readBytes]...)
	}

	if len(requestBytes) <= 0 {
		return
	}

	method, path, httpVersion, headers := parseHtmlRequest(requestBytes)

	req := request.New(method, path, httpVersion, headers)

	res := response.New(conn.Write)

	ep, err := s.matchEndpoint(req.Method, req.Path)

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

type serverConfig struct {
	ErrorHandler globalErrorHandler
}

func New(configs ...serverConfig) *server {
	config := serverConfig{
		ErrorHandler: func(req *request.Request, res *response.Response, err error) {
			err = res.JSON(response.JSONType{
				"error": err.Error(),
			})

			if err != nil {
				res.StatusCode(500)
				res.JSON(response.JSONType{
					"error": "Internal Server Error",
				})
			}

			res.Send()
		},
	}

	if len(configs) > 0 {
		config = configs[0]
	}

	return &server{
		errorHandler: config.ErrorHandler,
	}
}
