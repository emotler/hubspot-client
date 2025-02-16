package hubspot_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

var backoffSchedule = []time.Duration{
	1 * time.Second,
	3 * time.Second,
	10 * time.Second,
}

type Request struct {
	HSBaseUrl     string
	HSToken       string
	RequestMethod string
	UriEndPoint   string
	Params        url.Values
	JsonBody      *bytes.Reader
	Status        string
	StatusCode    int
	Headers       http.Header
	Content       []byte
	ContentLength int64
	Error         bool
	Message       string
}

func Init(hsBaseUrl string, hsToken string) Request {
	hs := Request{
		HSBaseUrl:     hsBaseUrl,
		HSToken:       hsToken,
		RequestMethod: "GET",
		Params:        url.Values{},
		JsonBody:      nil,
	}
	return hs
}

func (p *Request) Method(method string) *Request {
	p.RequestMethod = method
	return p
}

func (p *Request) EndPoint(endpoint string) *Request {
	p.UriEndPoint = endpoint
	return p
}

func (p *Request) QueryParams(queryParams map[string]string) *Request {
	for index, value := range queryParams {
		p.Params.Add(index, value)
	}
	return p
}

func (p *Request) Page(page int) *Request {
	p.Params.Add("page", strconv.Itoa(page))
	return p
}

func (p *Request) PageSize(pageSize int) *Request {
	p.Params.Add("pageSize", strconv.Itoa(pageSize))
	return p
}

func (p *Request) Properties(properties string) *Request {
	p.Params.Add("properties", properties)
	return p
}

func (p *Request) Associations(associations string) *Request {
	p.Params.Add("associations", associations)
	return p
}

func (p *Request) Json(body interface{}) *Request {
	marshalled, err := json.Marshal(body)
	if err != nil {
		log.Fatalf("impossible to marshall: %s", err)
	}
	p.JsonBody = bytes.NewReader(marshalled)
	return p
}

func (p *Request) GetStatus() int {
	return p.StatusCode
}

func (p *Request) Request() []byte {
	requestString := p.UriEndPoint
	var queryParams = p.Params.Encode()

	if len(queryParams) > 0 {
		requestString += "?" + queryParams
	}

	//fmt.Printf(requestString)
	req, err := GetRequest(p.RequestMethod, requestString, p.HSBaseUrl)
	if err != nil {
		fmt.Println(err)
	}
	if p.JsonBody != nil {
		req, err = BodyRequest(p.RequestMethod, requestString, p.HSBaseUrl, p.JsonBody)
		if err != nil {
			fmt.Println(err)
		}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.HSToken))
	client := &http.Client{}
	var response *http.Response

	for _, backoff := range backoffSchedule {
		response, err = client.Do(req)
		if err == nil {
			break
		}

		fmt.Fprintf(os.Stderr, "Request error: %+v\n", err)
		fmt.Fprintf(os.Stderr, "Retrying in %v\n", backoff)
		time.Sleep(backoff)
	}

	if err != nil {
		log.Fatalln(err)
	}

	p.Status = response.Status
	p.StatusCode = response.StatusCode
	p.ContentLength = response.ContentLength
	p.Headers = response.Header

	fmt.Println(p.Status)
	fmt.Println(p.StatusCode)
	fmt.Println(p.ContentLength)
	fmt.Println(p.RequestMethod)

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatalln(err)
		}
	}(response.Body)
	if err != nil {
		panic(err)
	}

	jsonDataFromHttp, err := io.ReadAll(response.Body)
	p.Content = jsonDataFromHttp
	if err != nil {
		panic(err)
	}

	return jsonDataFromHttp
}

func CreateRequest(requestString string, hsBaseUrl string) string {
	fmt.Println(fmt.Sprintf("%s/%s", hsBaseUrl, requestString))
	return fmt.Sprintf("%s/%s", hsBaseUrl, requestString)
}

func GetRequest(requestMethod string, requestString string, hsBaseUrl string) (*http.Request, error) {
	req, err := http.NewRequest(requestMethod, CreateRequest(requestString, hsBaseUrl), nil)
	return req, err
}

func BodyRequest(requestMethod string, requestString string, hsBaseUrl string, body *bytes.Reader) (*http.Request, error) {
	req, err := http.NewRequest(requestMethod, CreateRequest(requestString, hsBaseUrl), body)
	if err != nil {
		fmt.Println(err)
	}
	return req, err
}
