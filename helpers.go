package vesync

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	log "github.com/rs/zerolog/log"
	"github.com/golang/gddo/httputil/header"
)

var BaseUrl = "https://smartapi.vesync.com"

var DefaultRegion = "US"
var ApiTimeout = 5
var AppVersion = "2.5.1"
var PhoneBrand = "SM N9005"
var PhoneOs = "Android"
var MobileId = "1234567890123456"
var UserType = "1"
var BypassAppV = "VeSync 3.0.51"

func hashPassword(s string) string {
	hasher := md5.New()
	hasher.Write([]byte(s))
	return hex.EncodeToString(hasher.Sum(nil))
}
func ReqBodyBase(manager VeSync) map[string]interface{} {
	return map[string]interface{}{
		"timeZone": manager.TimeZone,
		"acceptLanguage": "en",
	}

}

func ReqBodyDetails() map[string]interface{} {
	return map[string]interface{}{
		"appVersion": AppVersion,
		"phoneBrand": PhoneBrand,
		"phoneOS": PhoneOs,
		"traceId": fmt.Sprint(time.Now().Unix()),
	}
}

func ReqBody(manager VeSync, type_ string) map[string]interface{} {
	var body = map[string]interface{}{}


	if type_ == "login" {
		for k, v := range ReqBodyBase(manager) { body[k] = v}
		for k, v := range ReqBodyDetails() { body[k] = v }

		body["email"] = manager.Username
		body["password"] = hashPassword(manager.Password)
		body["devToken"] = ""
		body["userType"] = UserType
		body["method"] = "login"
	}

	return body
}

func CallApi(api string, method string, headers map[string]interface{}, jsonMap map[string]interface{}, result *map[string]interface{}) (int, error) {
	var statusCode = 0
	var err error
	var requestBody *bytes.Buffer
	var req *http.Request
	var r *http.Response

	if jsonMap != nil && len(jsonMap) > 0 {
		postBody, _ := json.Marshal(jsonMap)
		requestBody = bytes.NewBuffer(postBody)
	} else {
		requestBody = bytes.NewBuffer([]byte{})
	}

	client := &http.Client{
		Timeout: time.Duration(ApiTimeout)*time.Second,
	}
	log.Debug().
		Str("Method", method).
		Str("Api", api).
		Msgf("[%s] calling '%s' api", method, api)


	req, err = http.NewRequest(strings.ToUpper(method), BaseUrl+ api, requestBody)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return statusCode, err
	}

	for k,v := range headers{
		req.Header.Add(k,v.(string))
	}

	r, err = client.Do(req)

	if err != nil {
		if r != nil{
			statusCode = r.StatusCode
		}
		defer r.Body.Close()
		return statusCode, err
	}

	if r.StatusCode == 200 {
		statusCode = 200
		err = decodeJSONBody(r, result)
	} else {
		log.Debug().
			Msgf("Unable to fetch %s%s", BaseUrl, api)
		err = errors.New(fmt.Sprintf("unable to fetch %s %s", BaseUrl, api))
	}

	defer r.Body.Close()
	return statusCode, err
}

type malformedRequest struct {
	status int
	msg    string
}

func (mr *malformedRequest) Error() string {
	return mr.msg
}

func decodeJSONBody(r *http.Response, dst interface{}) error {
	if r.Header.Get("Content-Type") != "" {
		value, _ := header.ParseValueAndParams(r.Header, "Content-Type")
		if value != "application/json" {
			msg := "Content-Type header is not application/json"
			return &malformedRequest{status: http.StatusUnsupportedMediaType, msg: msg}
		}
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(&dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case errors.Is(err, io.ErrUnexpectedEOF):
			msg := fmt.Sprintf("Request body contains badly-formed JSON")
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case err.Error() == "http: request body too large":
			msg := "Request body must not be larger than 1MB"
			return &malformedRequest{status: http.StatusRequestEntityTooLarge, msg: msg}

		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		msg := "Request body must only contain a single JSON object"
		return &malformedRequest{status: http.StatusBadRequest, msg: msg}
	}

	return nil
}

func CheckCode(r map[string]interface{}) bool {
	if r["code"].(float64) == 0{
		return true
	}
	return false
}