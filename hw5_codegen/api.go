package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
)

// вы можете использовать ApiError в коде, который получается в результате генерации
// считаем что это какая-то общеизвестная структура
type ApiError struct {
	HTTPStatus int
	Err        error
}

func (ae ApiError) Error() string {
	return ae.Err.Error()
}

// ----------------

const (
	statusUser      = 0
	statusModerator = 10
	statusAdmin     = 20
)

type MyApi struct {
	statuses map[string]int
	users    map[string]*User
	nextID   uint64
	mu       *sync.RWMutex
}

func NewMyApi() *MyApi {
	return &MyApi{
		statuses: map[string]int{
			"user":      0,
			"moderator": 10,
			"admin":     20,
		},
		users: map[string]*User{
			"rvasily": &User{
				ID:       42,
				Login:    "rvasily",
				FullName: "Vasily Romanov",
				Status:   statusAdmin,
			},
		},
		nextID: 43,
		mu:     &sync.RWMutex{},
	}
}

type ProfileParams struct {
	Login string `apivalidator:"required"`
}

type CreateParams struct {
	Login  string `apivalidator:"required,min=10"`
	Name   string `apivalidator:"paramname=full_name"`
	Status string `apivalidator:"enum=user|moderator|admin,default=user"`
	Age    int    `apivalidator:"min=0,max=128"`
}

type User struct {
	ID       uint64 `json:"id"`
	Login    string `json:"login"`
	FullName string `json:"full_name"`
	Status   int    `json:"status"`
}

type NewUser struct {
	ID uint64 `json:"id"`
}

// apigen:api {"url": "/user/profile", "auth": false}
func (srv *MyApi) Profile(ctx context.Context, in ProfileParams) (*User, error) {

	if in.Login == "bad_user" {
		return nil, fmt.Errorf("bad user")
	}

	srv.mu.RLock()
	user, exist := srv.users[in.Login]
	srv.mu.RUnlock()
	if !exist {
		return nil, ApiError{http.StatusNotFound, fmt.Errorf("user not exist")}
	}

	return user, nil
}

// apigen:api {"url": "/user/create", "auth": true, "method": "POST"}
func (srv *MyApi) Create(ctx context.Context, in CreateParams) (*NewUser, error) {
	if in.Login == "bad_username" {
		return nil, fmt.Errorf("bad user")
	}

	srv.mu.Lock()
	defer srv.mu.Unlock()

	_, exist := srv.users[in.Login]
	if exist {
		return nil, ApiError{http.StatusConflict, fmt.Errorf("user %s exist", in.Login)}
	}

	id := srv.nextID
	srv.nextID++
	srv.users[in.Login] = &User{
		ID:       id,
		Login:    in.Login,
		FullName: in.Name,
		Status:   srv.statuses[in.Status],
	}

	return &NewUser{id}, nil
}

// 2-я часть
// это похожая структура, с теми же методами, но у них другие параметры!
// код, созданный вашим кодогенератором работает с конкретной струткурой, про другие ничего не знает
// поэтому то что рядом есть ещё походая структура с такими же методами его нисколько не смущает

type OtherApi struct {
}

func NewOtherApi() *OtherApi {
	return &OtherApi{}
}

type OtherCreateParams struct {
	Username string `apivalidator:"required,min=3"`
	Name     string `apivalidator:"paramname=account_name"`
	Class    string `apivalidator:"enum=warrior|sorcerer|rouge,default=warrior"`
	Level    int    `apivalidator:"min=1,max=50"`
}

type OtherUser struct {
	ID       uint64 `json:"id"`
	Login    string `json:"login"`
	FullName string `json:"full_name"`
	Level    int    `json:"level"`
}

// apigen:api {"url": "/user/create", "auth": true, "method": "POST"}
func (srv *OtherApi) Create(ctx context.Context, in OtherCreateParams) (*OtherUser, error) {
	return &OtherUser{
		ID:       12,
		Login:    in.Username,
		FullName: in.Name,
		Level:    in.Level,
	}, nil
}

//==================================
/*`handler$methodName` - обёртка над методом структуры `$methodName`
- осуществляет все проверки, выводит ошибки или результат в формате JSON*/
/*
* метод (POST)
* авторизация
* параметры в порядке следования в структуре
Авторизация проверяется просто на то что в хедере пришло значение `100500`
*/
//
//func (srv *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//	switch r.URL.Path {
//	case "/user/create":
//		srv.handlerCreate(w, r)
//	default:
//		apiErr := &ApiError{http.StatusNotFound, errors.New("unknown method")}
//		apiErr.WriteResponse(w)
//	}
//}
//
//func (srv *OtherApi) handlerCreate(w http.ResponseWriter, r *http.Request) {
//	methodIsBad := CheckMethod("POST", r.Method)
//	if methodIsBad != nil {
//		methodIsBad.WriteResponse(w)
//		return
//	}
//
//	/*	var in *OtherCreateParams
//		var err *ApiError*/
//	in, err := SetOtherCreateParams(r)
//	if err != nil {
//		err.WriteResponse(w)
//		return
//	}
//
//	/*	var usr *OtherUser
//		var methodErr error*/
//	ctx := r.Context()
//	usr, methodErr := srv.Create(ctx, *in)
//	if methodErr != nil {
//		var ae *ApiError
//		ae = handleError(methodErr)
//		ae.WriteResponse(w)
//		return
//		/*		ae := &ApiError{http.StatusBadRequest, errMethod}
//				ae.WriteResponse(w)
//				return*/
//	}
//
//	isOk := &ApiAnswer{http.StatusOK, usr}
//	isOk.WriteResponse(w)
//}
//
//// OtherCreateParams set/validate
//func SetOtherCreateParams(r *http.Request) (*OtherCreateParams, *ApiError) {
//	var parentStruct OtherCreateParams
//
//	vals, setErr := SetValues(r)
//	if setErr != nil {
//		return nil, &ApiError{http.StatusInternalServerError, setErr}
//	}
//
//	username := vals.Get("username")
//	/* required*/
//	if username == "" {
//		return nil, &ApiError{http.StatusBadRequest, errors.New("username must me not empty")}
//	}
//	/* min, max for string*/
//	if len(username) < 3 {
//		return nil, &ApiError{http.StatusBadRequest, errors.New("username len must be >= 3")}
//	}
//	parentStruct.Username = username
//
//	account_name := vals.Get("account_name")
//	/* min, max for string*/
//	parentStruct.Name = account_name
//
//	class := vals.Get("class")
//	/* min, max for string*/
//	/*enum*/
//	classEnum := []string{"warrior", "sorcerer", "rouge"}
//	if class == "" {
//		class = "warrior"
//	} else {
//		ok := false
//		for _, v := range classEnum {
//			if class == v {
//				ok = true
//				break
//			}
//		}
//		if !ok {
//			return nil, &ApiError{http.StatusBadRequest, errors.New("class must be one of [" + strings.Join(classEnum, ", ") + "]")}
//		}
//	}
//	parentStruct.Class = class
//
//	level := vals.Get("level")
//	/* atoi, min, max for int*/
//	level0, err := strconv.Atoi(level)
//	if err != nil {
//		return nil, &ApiError{http.StatusBadRequest, errors.New("level must be int")}
//	}
//	if level0 < 1 {
//		return nil, &ApiError{http.StatusBadRequest, errors.New("level must be >= 1")}
//	}
//	if level0 > 50 {
//		return nil, &ApiError{http.StatusBadRequest, errors.New("level must be <= 50")}
//	}
//	parentStruct.Level = level0
//	return &parentStruct, nil
//}
//
//func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//	switch r.URL.Path {
//	case "/user/create":
//		srv.handlerCreate(w, r)
//	case "/user/profile":
//		srv.handlerProfile(w, r)
//	default:
//		apiErr := &ApiError{http.StatusNotFound, errors.New("unknown method")}
//		apiErr.WriteResponse(w)
//	}
//}
//
///*`handler$methodName` - обёртка над методом структуры `$methodName`
//- осуществляет все проверки, выводит ошибки или результат в формате JSON*/
///*
//* метод (POST)
//* авторизация
//* параметры в порядке следования в структуре
//Авторизация проверяется просто на то что в хедере пришло значение `100500`
//*/
//func (srv *MyApi) handlerCreate(w http.ResponseWriter, r *http.Request) {
//	// apigen:api {"url": "/user/create", "auth": true, "method": "POST"}
//	methodIsBad := CheckMethod("POST", r.Method)
//	if methodIsBad != nil {
//		methodIsBad.WriteResponse(w)
//		return
//	}
//
//	isUnauthorized := CheckAuthorization(r.Header.Get("X-Auth"))
//	if isUnauthorized != nil {
//		isUnauthorized.WriteResponse(w)
//		return
//	}
//
//	var in *CreateParams
//	var err *ApiError
//	in, err = SetCreateParams(r)
//	if err != nil {
//		err.WriteResponse(w)
//		return
//	}
//
//	/*	var usr *NewUser
//		var methodErr error*/
//	ctx := r.Context()
//	usr, methodErr := srv.Create(ctx, *in)
//	if methodErr != nil {
//		var ae *ApiError
//		ae = handleError(methodErr)
//		ae.WriteResponse(w)
//		return
//		/*		ae := &ApiError{http.StatusBadRequest, errMethod}
//				ae.WriteResponse(w)
//				return*/
//	}
//
//	isOk := &ApiAnswer{http.StatusOK, usr}
//	isOk.WriteResponse(w)
//}
//
//func (srv *MyApi) handlerProfile(w http.ResponseWriter, r *http.Request) {
//	// apigen:api {"url": "/user/profile", "auth": false}
//	//func (srv *MyApi) Profile(ctx context.Context, in ProfileParams) (*User, error)
//
//	var in *ProfileParams
//	var err *ApiError
//	in, err = SetProfileParams(r)
//	if err != nil {
//		err.WriteResponse(w)
//		return
//	}
//
//	/*	var usr *User
//		var methodErr error*/
//	ctx := r.Context()
//	usr, methodErr := srv.Profile(ctx, *in)
//	if methodErr != nil {
//		var ae *ApiError
//		ae = handleError(methodErr)
//		ae.WriteResponse(w)
//		return
//	}
//
//	isOk := &ApiAnswer{http.StatusOK, usr}
//	isOk.WriteResponse(w)
//}
//
//// -- Общие методы, учитывать
//func CheckMethod(allowed, current string) *ApiError {
//	if allowed == current {
//		return nil
//	}
//
//	return &ApiError{http.StatusNotAcceptable,
//		errors.New(`bad method`),
//	}
//}
//
//const isAuthorized = "100500"
//
//func CheckAuthorization(s string) *ApiError {
//	if s == isAuthorized {
//		return nil
//	}
//	return &ApiError{http.StatusForbidden,
//		errors.New(`unauthorized`),
//	}
//}
//
//type ApiAnswer struct {
//	HTTPStatus int
//	msg        any
//}
//
//func (answ ApiError) toJson() ([]byte, error) {
//	resp := make(map[string]string)
//	resp["error"] = answ.Error()
//	jsonResp, err := json.Marshal(resp)
//	if err != nil {
//		return nil, errors.Errorf("Error happened in JSON marshal. Err: %s", err)
//	}
//	return jsonResp, nil
//}
//
//func (answ ApiError) WriteResponse(w http.ResponseWriter) {
//	msg, err := answ.toJson()
//	if err != nil {
//		txt := fmt.Sprintf("Error happened in http.ResponseWriter write. Err: %s", err)
//		errInner := &ApiError{http.StatusInternalServerError, errors.New(txt)}
//		errInner.WriteResponse(w)
//		log.Fatalf(txt)
//	} else {
//		w.WriteHeader(answ.HTTPStatus)
//		w.Header().Set("Content-Type", "application/json")
//	}
//	w.Write(msg)
//}
//
//func (answ *ApiAnswer) toJson() ([]byte, error) {
//	resp := make(map[string]any)
//	resp["response"] = answ.msg
//	resp["error"] = ""
//	jsonResp, err := json.Marshal(resp)
//	if err != nil {
//		return nil, errors.Errorf("Error happened in JSON marshal. Err: %s", err)
//	}
//	return jsonResp, nil
//}
//
//func (answ *ApiAnswer) WriteResponse(w http.ResponseWriter) {
//	msg, err := answ.toJson()
//	if err != nil {
//		txt := fmt.Sprintf("Error happened in http.ResponseWriter write. Err: %s", err)
//		errInner := &ApiError{http.StatusInternalServerError, errors.New(txt)}
//		errInner.WriteResponse(w)
//		log.Fatalf(txt)
//	} else {
//		w.WriteHeader(answ.HTTPStatus)
//		w.Header().Set("Content-Type", "application/json")
//	}
//	w.Write(msg)
//}
//
//func SetValues(r *http.Request) (*url.Values, *ApiError) {
//	var vals url.Values
//	switch r.Method {
//	case http.MethodGet:
//		vals = r.URL.Query()
//		return &vals, nil
//	case http.MethodPost:
//		var body []byte
//		var err error
//		body, err = io.ReadAll(r.Body)
//		if err != nil {
//			return nil, &ApiError{http.StatusInternalServerError, err}
//		}
//		vals, err = url.ParseQuery(string(body))
//		if err != nil {
//			return nil, &ApiError{http.StatusInternalServerError, err}
//		}
//		return &vals, nil
//	default:
//		return nil, &ApiError{http.StatusMethodNotAllowed, errors.New("unknown method")}
//	}
//}
//
//func handleError(methodErr error) *ApiError {
//	if reflect.TypeOf(methodErr) == reflect.TypeOf(ApiError{}) {
//		ae, ok := methodErr.(ApiError)
//		if !ok {
//			innerErr := ApiError{http.StatusInternalServerError,
//				errors.Errorf("error converting %v to ApiError", methodErr)}
//			return &innerErr
//		}
//		return &ae
//	} else {
//		return &ApiError{http.StatusInternalServerError, methodErr}
//	}
//}
//
////-------
