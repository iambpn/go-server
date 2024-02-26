package response

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

var httpVersion = "HTTP/1.1"
var statusCodeToText = map[string]string{
	"1xx": "Informational",
	"100": "Continue",
	"101": "Switching Protocols",
	"102": "Processing",
	"103": "Early Hints",
	"2xx": "Success",
	"200": "OK",
	"201": "Created",
	"202": "Accepted",
	"203": "Non-Authoritative Information",
	"204": "No Content",
	"205": "Reset Content",
	"206": "Partial Content",
	"207": "Multi-Status",
	"208": "Already Reported",
	"226": "IM Used",
	"3xx": "Redirection",
	"300": "Multiple Choices",
	"301": "Moved Permanently",
	"302": "Found (or Moved Temporarily)",
	"303": "See Other",
	"304": "Not Modified",
	"305": "Use Proxy",
	"306": "(Unused)",
	"307": "Temporary Redirect",
	"308": "Permanent Redirect",
	"4xx": "Client Side Error",
	"400": "Bad Request",
	"401": "Unauthorized",
	"402": "Payment Required",
	"403": "Forbidden",
	"404": "Not Found",
	"405": "Method Not Allowed",
	"406": "Not Acceptable",
	"407": "Proxy Authentication Required",
	"408": "Request Timeout",
	"409": "Conflict",
	"410": "Gone",
	"411": "Length Required",
	"412": "Precondition Failed",
	"413": "Payload Too Large",
	"414": "URI Too Long",
	"415": "Unsupported Media Type",
	"416": "Range Not Satisfiable",
	"417": "Expectation Failed",
	"418": "I'm a teapot",
	"421": "Misdirected Request",
	"422": "Unprocessable Entity",
	"423": "Locked",
	"424": "Failed Dependency",
	"425": "Too Early",
	"426": "Upgrade Required",
	"428": "Precondition Required",
	"429": "Too Many Requests",
	"431": "Request Header Fields Too Large",
	"451": "Unavailable For Legal Reasons",
	"5xx": "Server Error",
	"500": "Internal Server Error",
	"501": "Not Implemented",
	"502": "Bad Gateway",
	"503": "Service Unavailable",
	"504": "Gateway Timeout",
	"505": "HTTP Version Not Supported",
	"506": "Variant Also Negotiates",
	"507": "Insufficient Storage",
	"508": "Loop Detected",
	"510": "Not Extended",
	"511": "Network Authentication Required",
}

/*
Response Type
*/
type Response struct {
	Headers    map[string]string
	writer     func(b []byte) (n int, err error)
	statusCode int
	content    []byte
}

func New(writer func(b []byte) (n int, err error)) *Response {
	headers := make(map[string]string)

	headers["Date"] = time.Now().UTC().String()
	headers["Server"] = "go server by @iambpn"
	headers["Content-Type"] = "plain/text"
	headers["Content-Length"] = "0"

	return &Response{
		Headers:    headers,
		statusCode: 200,
		writer:     writer,
	}
}

func (res *Response) SetHeader(key string, value string) {
	res.Headers[key] = value
}

func getHumanReadableStatusCode(statusCode int) string {
	if status, ok := statusCodeToText[fmt.Sprint(statusCode)]; ok {
		return status
	}

	defaultCode := string(fmt.Sprint(statusCode)) + "xx"
	return statusCodeToText[defaultCode]
}

func (res *Response) Send() {
	lines := []string{}

	lines = append(lines, fmt.Sprintf("%s %d %s", httpVersion, res.statusCode, getHumanReadableStatusCode(res.statusCode)))

	for key, value := range res.Headers {
		lines = append(lines, fmt.Sprintf("%s: %s", key, value))
	}

	lines = append(lines, "\n")

	content := []byte(strings.Join(lines, "\n"))
	content = append(content, res.content...)

	res.writer(content)
}

type JSONType = map[string]interface{}

func (res *Response) JSON(message JSONType) error {
	data, err := json.Marshal(message)

	if err != nil {
		return err
	}

	res.content = data
	res.Headers["Content-Type"] = "application/json"
	res.Headers["Content-Length"] = fmt.Sprint(len(data))
	return nil
}

func (res *Response) StatusCode(code int) {
	res.statusCode = code
}
