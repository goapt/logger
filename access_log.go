package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	MIMEJSON     = "application/json"
	MIMEPOSTForm = "application/x-www-form-urlencoded"
)

type ResponseWriter interface {
	http.ResponseWriter
	Status() int
	Body() []byte
}

var defaultMaxLength = 50000

type AccessEntry struct {
	StartTime      time.Time
	request        *http.Request
	body           []byte
	Response       ResponseWriter
	LogInfo        map[string]interface{}
	URIFilter      []string // 忽略的URI
	isSkip         bool
	RequestFilter  func(body []byte, r *http.Request) []byte // 过滤request_body敏感信息
	ResponseFilter func(body []byte, r *http.Request) []byte // 过滤response_body敏感信息
}

func NewAccessLog(req *http.Request) *AccessEntry {
	entry := &AccessEntry{
		StartTime: time.Now(),
		request:   req,
	}

	// 跳过健康检查等需要忽略记录的日志
	entry.URIFilter = append(entry.URIFilter, "/heartbeat/check", "/metrics")
	if inSlice(req.RequestURI, entry.URIFilter) {
		entry.isSkip = true
	}

	if !entry.isSkip && req.Body != nil {
		// 由业务决定过滤记录的日志
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return nil
		}
		req.Body = ioutil.NopCloser(bytes.NewReader(body))
		entry.body = body
	}
	return entry
}

func (l *AccessEntry) Get() map[string]interface{} {
	if l.isSkip {
		return nil
	}

	if l.RequestFilter != nil {
		l.body = l.RequestFilter(l.body, l.request)
	}

	// 部分业务返回数据需要脱敏
	var respBody []byte
	var httpStatus int
	var responseHeader string
	if l.Response != nil {
		respBody = l.Response.Body()
		httpStatus = l.Response.Status()
		if l.ResponseFilter != nil {
			respBody = l.ResponseFilter(respBody, l.request)
		}

		responseHeader = jsonEncode(queryToMap(l.Response.Header()))
	}

	endTime := time.Now()
	hostname, _ := os.Hostname()
	logData := map[string]interface{}{
		"request_id":       l.request.Header.Get("X-Request-ID"),
		"request_method":   l.request.Method,
		"request_header":   jsonEncode(queryToMap(l.request.Header)),
		"request_uri":      l.request.URL.Path,
		"request_time":     l.StartTime.Format("2006-01-02 15:04:05"),
		"response_time":    endTime.Format("2006-01-02 15:04:05"),
		"request_duration": fmt.Sprintf("%.3f", endTime.Sub(l.StartTime).Seconds()),
		"request_query":    jsonEncode(queryToMap(l.request.URL.Query())),
		"request_body":     string(l.body),
		"response_header":  responseHeader,
		"response_body":    string(respBody),
		"http_user_agent":  l.request.UserAgent(),
		"http_status":      httpStatus,
		"host_name":        hostname,
		"server_name":      l.request.Host,
		"remote_addr":      remoteAddr(l.request),
		"proto":            l.request.Proto,
		"info":             "",
	}

	if l.LogInfo != nil {
		logData["info"] = jsonEncode(l.LogInfo)
	}

	if strings.HasPrefix(l.request.Header.Get("Content-Type"), MIMEPOSTForm) {
		requestData, err := parseRequest(l.body, l.request)
		if err == nil {
			logData["request_body_raw"] = logData["request_body"]
			logData["request_body"] = jsonEncode(requestData)
		}
	}

	// 超长处理
	hasTruncate := ""
	for k, v := range logData {
		if vv, ok := v.(string); ok {
			if len(vv) > defaultMaxLength {
				hasTruncate = "Y"
				logData[k] = vv[0:defaultMaxLength] + "...(truncate)"
			}
		}
	}

	if hasTruncate != "" {
		logData["has_truncate"] = hasTruncate
	}

	return logData
}

func remoteAddr(r *http.Request) string {
	var ip string
	if ips := r.Header.Get("X-Forwarded-For"); ips != "" {
		ipSli := strings.Split(ips, ",")
		for _, v := range ipSli {
			v = strings.TrimSpace(v)
			switch {
			case v == "":
				continue
			case v == "unknow":
				continue
			case v == "127.0.0.1":
				continue
			case strings.HasPrefix(v, "10."):
				continue
			case strings.HasPrefix(v, "172"):
				continue
			case strings.HasPrefix(v, "192"):
				continue
			}

			return v
		}
	} else if ip = r.Header.Get("Client-Ip"); ip != "" {
		return strings.TrimSpace(ip)
	} else if ip = r.Header.Get("Remote-Addr"); ip != "" {
		return strings.TrimSpace(ip)
	}

	if ip != "" {
		return ip
	}

	return "-1"
}

func jsonEncode(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func inSlice(uri string, s []string) bool {
	for _, v := range s {
		if uri == v || strings.HasSuffix(uri, v) {
			return true
		}
	}
	return false
}

func queryToMap(query map[string][]string) map[string]interface{} {
	qm := make(map[string]interface{})
	for k, v := range query {
		if len(v) == 1 {
			qm[k] = v[0]
		} else {
			qm[k] = v
		}
	}
	return qm
}

func parseRequest(body []byte, r *http.Request) (map[string]interface{}, error) {
	switch contentType(r) {
	case MIMEJSON:
		return parseJson(body)
	case MIMEPOSTForm:
		return parseForm(r)
	default:
		return nil, errors.New("unsupported content type")
	}
}

// parseJson json into map
func parseJson(body []byte) (map[string]interface{}, error) {
	requestData := make(map[string]interface{})
	// @help 反序列化JSON的数字类型至interface{}的时候，不要直接转换成float64的科学计数法，而是转换成json.Number类型
	// 如果自动转成科学技术法，会导致签名计算失败，试想：10000000=>1.0e7
	// @link https://ethancai.github.io/2016/06/23/bad-parts-about-json-serialization-in-Golang/
	decoder := json.NewDecoder(bytes.NewBuffer(body))
	decoder.UseNumber()
	if err := decoder.Decode(&requestData); err != nil {
		return nil, err
	}
	return requestData, nil
}

// parseForm form into map
func parseForm(r *http.Request) (map[string]interface{}, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	requestData := make(map[string]interface{})

	for k, v := range r.Form {
		requestData[k] = v[0] // take first
	}
	return requestData, nil
}

func filterFlags(content string) string {
	for i, char := range content {
		if char == ' ' || char == ';' {
			return content[:i]
		}
	}
	return content
}

func contentType(r *http.Request) string {
	return filterFlags(r.Header.Get("Content-Type"))
}
