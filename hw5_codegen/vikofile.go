package main

import (
	"encoding/json"
	"fmt"
	errors "github.com/pkg/errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// ProfileParams set/validate
func SetProfileParams(r *http.Request) (*ProfileParams, *ApiError) {
	var parentStruct ProfileParams
	vals, setErr := SetValues(r)
	if setErr != nil {
		return nil, setErr
	}

	login := vals.Get("login")
	/* required*/
	if login == "" {
		return nil, &ApiError{http.StatusBadRequest, errors.New("login must me not empty")}
	}
	/* min, max for string*/
	parentStruct.Login = login
	return &parentStruct, nil
}

// CreateParams set/validate
func SetCreateParams(r *http.Request) (*CreateParams, *ApiError) {
	var parentStruct CreateParams
	vals, setErr := SetValues(r)
	if setErr != nil {
		return nil, setErr
	}

	login := vals.Get("login")
	/* required*/
	if login == "" {
		return nil, &ApiError{http.StatusBadRequest, errors.New("login must me not empty")}
	}
	/* min, max for string*/
	if len(login) < 10 {
		return nil, &ApiError{http.StatusBadRequest, errors.New("login len must be >= 10")}
	}
	parentStruct.Login = login

	full_name := vals.Get("full_name")
	/* min, max for string*/
	parentStruct.Name = full_name

	status := vals.Get("status")
	/* min, max for string*/
	/*enum*/
	statusEnum := []string{"user", "moderator", "admin"}
	if status == "" {
		status = "user"
	} else {
		ok := false
		for _, v := range statusEnum {
			if status == v {
				ok = true
				break
			}
		}
		if !ok {
			return nil, &ApiError{http.StatusBadRequest, errors.New("status must be one of [" + strings.Join(statusEnum, ", ") + "]")}
		}
	}
	parentStruct.Status = status

	age := vals.Get("age")
	/* atoi, min, max for int*/
	age0, err := strconv.Atoi(age)
	if err != nil {
		return nil, &ApiError{http.StatusBadRequest, errors.New("age must be int")}
	}
	if age0 < 0 {
		return nil, &ApiError{http.StatusBadRequest, errors.New("age must be >= 0")}
	}
	if age0 > 128 {
		return nil, &ApiError{http.StatusBadRequest, errors.New("age must be <= 128")}
	}
	parentStruct.Age = age0
	return &parentStruct, nil
}

// OtherCreateParams set/validate
func SetOtherCreateParams(r *http.Request) (*OtherCreateParams, *ApiError) {
	var parentStruct OtherCreateParams
	vals, setErr := SetValues(r)
	if setErr != nil {
		return nil, setErr
	}

	username := vals.Get("username")
	/* required*/
	if username == "" {
		return nil, &ApiError{http.StatusBadRequest, errors.New("username must me not empty")}
	}
	/* min, max for string*/
	if len(username) < 3 {
		return nil, &ApiError{http.StatusBadRequest, errors.New("username len must be >= 3")}
	}
	parentStruct.Username = username

	account_name := vals.Get("account_name")
	/* min, max for string*/
	parentStruct.Name = account_name

	class := vals.Get("class")
	/* min, max for string*/
	/*enum*/
	classEnum := []string{"warrior", "sorcerer", "rouge"}
	if class == "" {
		class = "warrior"
	} else {
		ok := false
		for _, v := range classEnum {
			if class == v {
				ok = true
				break
			}
		}
		if !ok {
			return nil, &ApiError{http.StatusBadRequest, errors.New("class must be one of [" + strings.Join(classEnum, ", ") + "]")}
		}
	}
	parentStruct.Class = class

	level := vals.Get("level")
	/* atoi, min, max for int*/
	level0, err := strconv.Atoi(level)
	if err != nil {
		return nil, &ApiError{http.StatusBadRequest, errors.New("level must be int")}
	}
	if level0 < 1 {
		return nil, &ApiError{http.StatusBadRequest, errors.New("level must be >= 1")}
	}
	if level0 > 50 {
		return nil, &ApiError{http.StatusBadRequest, errors.New("level must be <= 50")}
	}
	parentStruct.Level = level0
	return &parentStruct, nil
}

// MyApi serveHTTP
func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/profile":
		srv.handlerProfile(w, r)
	case "/user/create":
		srv.handlerCreate(w, r)
	default:
		apiErr := &ApiError{http.StatusNotFound, errors.New("unknown method")}
		apiErr.WriteResponse(w)
	}
}
func (srv *MyApi) handlerProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	in, err := SetProfileParams(r)
	if err != nil {
		err.WriteResponse(w)
		return
	}
	usr, methodErr := srv.Profile(ctx, *in)
	if methodErr != nil {
		var ae *ApiError
		ae = handleError(methodErr)
		ae.WriteResponse(w)
		return
	}
	isOk := &ApiAnswer{http.StatusOK, usr}
	isOk.WriteResponse(w)

}
func (srv *MyApi) handlerCreate(w http.ResponseWriter, r *http.Request) {
	methodIsBad := CheckMethod("POST", r.Method)
	if methodIsBad != nil {
		methodIsBad.WriteResponse(w)
		return
	}
	isUnauthorized := CheckAuthorization(r.Header.Get("X-Auth"))
	if isUnauthorized != nil {
		isUnauthorized.WriteResponse(w)
		return
	}
	ctx := r.Context()
	in, err := SetCreateParams(r)
	if err != nil {
		err.WriteResponse(w)
		return
	}
	usr, methodErr := srv.Create(ctx, *in)
	if methodErr != nil {
		var ae *ApiError
		ae = handleError(methodErr)
		ae.WriteResponse(w)
		return
	}
	isOk := &ApiAnswer{http.StatusOK, usr}
	isOk.WriteResponse(w)

}

// OtherApi serveHTTP
func (srv *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/create":
		srv.handlerCreate(w, r)
	default:
		apiErr := &ApiError{http.StatusNotFound, errors.New("unknown method")}
		apiErr.WriteResponse(w)
	}
}
func (srv *OtherApi) handlerCreate(w http.ResponseWriter, r *http.Request) {
	methodIsBad := CheckMethod("POST", r.Method)
	if methodIsBad != nil {
		methodIsBad.WriteResponse(w)
		return
	}
	isUnauthorized := CheckAuthorization(r.Header.Get("X-Auth"))
	if isUnauthorized != nil {
		isUnauthorized.WriteResponse(w)
		return
	}
	ctx := r.Context()
	in, err := SetOtherCreateParams(r)
	if err != nil {
		err.WriteResponse(w)
		return
	}
	usr, methodErr := srv.Create(ctx, *in)
	if methodErr != nil {
		var ae *ApiError
		ae = handleError(methodErr)
		ae.WriteResponse(w)
		return
	}
	isOk := &ApiAnswer{http.StatusOK, usr}
	isOk.WriteResponse(w)

}
func CheckMethod(allowed, current string) *ApiError {
	if allowed == current {
		return nil
	}

	return &ApiError{http.StatusNotAcceptable,
		errors.New(`bad method`),
	}
}

const isAuthorized = "100500"

func CheckAuthorization(s string) *ApiError {
	if s == isAuthorized {
		return nil
	}
	return &ApiError{http.StatusForbidden,
		errors.New(`unauthorized`),
	}
}

type ApiAnswer struct {
	HTTPStatus int
	msg        any
}

func (answ ApiError) toJson() ([]byte, error) {
	resp := make(map[string]string)
	resp["error"] = answ.Error()
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		return nil, errors.Errorf("Error happened in JSON marshal. Err: %s", err)
	}
	return jsonResp, nil
}

func (answ ApiError) WriteResponse(w http.ResponseWriter) {
	msg, err := answ.toJson()
	if err != nil {
		txt := fmt.Sprintf("Error happened in http.ResponseWriter write. Err: %s", err)
		errInner := &ApiError{http.StatusInternalServerError, errors.New(txt)}
		errInner.WriteResponse(w)
		log.Fatalf(txt)
	} else {
		w.WriteHeader(answ.HTTPStatus)
		w.Header().Set("Content-Type", "application/json")
	}
	w.Write(msg)
}

func (answ *ApiAnswer) toJson() ([]byte, error) {
	resp := make(map[string]any)
	resp["response"] = answ.msg
	resp["error"] = ""
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		return nil, errors.Errorf("Error happened in JSON marshal. Err: %s", err)
	}
	return jsonResp, nil
}

func (answ *ApiAnswer) WriteResponse(w http.ResponseWriter) {
	msg, err := answ.toJson()
	if err != nil {
		txt := fmt.Sprintf("Error happened in http.ResponseWriter write. Err: %s", err)
		errInner := &ApiError{http.StatusInternalServerError, errors.New(txt)}
		errInner.WriteResponse(w)
		log.Fatalf(txt)
	} else {
		w.WriteHeader(answ.HTTPStatus)
		w.Header().Set("Content-Type", "application/json")
	}
	w.Write(msg)
}

func SetValues(r *http.Request) (*url.Values, *ApiError) {
	var vals url.Values
	switch r.Method {
	case http.MethodGet:
		vals = r.URL.Query()
		return &vals, nil
	case http.MethodPost:
		var body []byte
		var err error
		body, err = io.ReadAll(r.Body)
		if err != nil {
			return nil, &ApiError{http.StatusInternalServerError, err}
		}
		vals, err = url.ParseQuery(string(body))
		if err != nil {
			return nil, &ApiError{http.StatusInternalServerError, err}
		}
		return &vals, nil
	default:
		return nil, &ApiError{http.StatusMethodNotAllowed, errors.New("unknown method")}
	}
}

func handleError(methodErr error) *ApiError {
	if reflect.TypeOf(methodErr) == reflect.TypeOf(ApiError{}) {
		ae, ok := methodErr.(ApiError)
		if !ok {
			innerErr := ApiError{http.StatusInternalServerError,
				errors.Errorf("error converting %v to ApiError", methodErr)}
			return &innerErr
		}
		return &ae
	} else {
		return &ApiError{http.StatusInternalServerError, methodErr}
	}
}
