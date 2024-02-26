package request

/*
Request Type
*/
type Request struct {
	Method      string
	Params      map[string]string
	query       map[string]string
	Path        string
	Body        interface{}
	HttpVersion string
	Headers     map[string]string
}

func New(method, path, httpVersion string, headers map[string]string) *Request {
	return &Request{
		Method:      method,
		Path:        path,
		HttpVersion: httpVersion,
		Headers:     headers,
	}
}
