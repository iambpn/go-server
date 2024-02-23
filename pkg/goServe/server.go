package goServe

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

/*
Request Type
*/
type request struct {
	Method      string
	Params      map[string]string
	query       map[string]string
	Path        string
	Body        interface{}
	HttpVersion string
	Headers     map[string]string
}

/*
Response Type
*/
type response struct {
	Headers map[string]string
}

type endpoint struct {
	method string
	path   string
	cb     func(request, response)
}

type server struct {
	listener  net.Listener
	endpoints []endpoint
}

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

		go s.processRequest(conn)
	}
}

func (s *server) processRequest(conn net.Conn) error {
	defer conn.Close()

	buffer := make([]byte, 1024)
	requestBytes := []byte{}

	for {
		// set timeout for 1ms
		err := conn.SetReadDeadline(time.Now().Add(time.Millisecond))

		if err != nil {
			return err
		}

		readBytes, err := conn.Read(buffer)

		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				break
			}

			return err
		}

		requestBytes = append(requestBytes, buffer[:readBytes]...)
	}

	if len(requestBytes) <= 0 {
		return nil
	}

	method, path, httpVersion, headers := s.parseHtmlRequest(requestBytes)

	req := request{
		Method:      method,
		Path:        path,
		HttpVersion: httpVersion,
		Headers:     headers,
	}
}

func (s *server) parseHtmlRequest(requestBytes []byte) (method string, path string, httpVersion string, headers map[string]string) {
	headerLines := strings.Split(string(requestBytes), "\n")

	// parse 1st line and implement different methods
	metadata := strings.Split(strings.TrimSpace(headerLines[0]), " ")

	method = metadata[0]
	path = metadata[1]
	httpVersion = metadata[2]

	if len(headerLines) > 1 {
		headers = s.parseHeaders(headerLines[1:])
	}

	return method, path, httpVersion, headers
}

func (s *server) parseHeaders(headers []string) map[string]string {
	headersMap := make(map[string]string)

	for _, val := range headers {
		if val = strings.TrimSpace(val); val != "" {
			keyVal := strings.Split(val, ": ")
			headersMap[keyVal[0]] = strings.Join(keyVal[1:], ": ")
		}
	}

	return headersMap
}

func (s *server) findEndpoint(){
	for _, ep := range s.endpoints {
		switch strings.ToLower(ep.method) {
		case "get":
		case "post":
		case "put":
		case "patch":
		case "delete":
		default:
			
		}
	}
}

/*
method is the standard REST Method
path is the app endpoints
handler is the endpoint handler
*/
func (s *server) AddPath(method string, path string, handler func(request, response)) {
	s.endpoints = append(s.endpoints, endpoint{method, path, handler})
}

func New() *server {
	return &server{}
}
