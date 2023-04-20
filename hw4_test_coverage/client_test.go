package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	idTg         = "id"
	fNameTg      = "first_name"
	lNameTg      = "last_name"
	ageTg        = "age"
	aboutTg      = "about"
	gndrTg       = "gender"
	usrTg        = "row"
	nameTg       = "name"
	allowedOF    = []string{idTg, ageTg, nameTg}
	allowedOrder = []int{OrderByAsc, OrderByAsIs, OrderByDesc}
	limitF       = "limit"
	offsetF      = "offset"
	queryF       = "query"
	order_fieldF = "order_field"
	order_byF    = "order_by"
	fields       = []string{limitF, offsetF, queryF, order_fieldF, order_byF}
)

func (req *SearchRequest) search(data []User) []User {
	if req.Query == "" {
		return data
	}

	var res []User
	for _, usr := range data {
		n := strings.ToLower(usr.Name)
		ab := strings.ToLower(usr.About)
		if strings.Contains(n, req.Query) || strings.Contains(ab, req.Query) {
			res = append(res, usr)
		}
	}

	return res
}

func TestSearch(t *testing.T) {
	s := SearchRequest{0, 0, "kt0", "", 0}
	data := []User{
		{30, "a", 33, "b", "c"},
		{17, "kt0", 12, "c", "r"},
		{33, "a", 33, "kakT0TAk", "c"},
	}
	res := s.search(data)
	right := []User{
		{17, "kt0", 12, "c", "r"},
		{33, "a", 33, "kakT0TAk", "c"},
	}
	if !reflect.DeepEqual(res, right) {
		t.Errorf("1 - wrong result, expected %#v, got %#v", right, res)
	}

	s.Query = ""
	res = s.search(data)
	right = data
	if !reflect.DeepEqual(res, right) {
		t.Errorf("2 - wrong result, expected %#v, got %#v", right, res)
	}

}

func (req *SearchRequest) sort(data []User) []User {
	if req.OrderBy == OrderByAsIs {
		return data
	}

	switch req.OrderField {
	case idTg:
		data = req.sortId(data)
	case nameTg:
		data = req.sortName(data)
	case ageTg:
		data = req.sortAge(data)
	}
	return data
}

func (req *SearchRequest) sortId(data []User) []User {
	if req.OrderBy == OrderByAsc {
		sort.Slice(data, func(i, j int) bool { return data[i].Id < data[j].Id })
	} else if req.OrderBy == OrderByDesc {
		sort.Slice(data, func(i, j int) bool { return data[i].Id > data[j].Id })
	}
	return data
}

func (req *SearchRequest) sortName(data []User) []User {
	if req.OrderBy == OrderByAsc {
		sort.Slice(data, func(i, j int) bool { return data[i].Name < data[j].Name })
	} else if req.OrderBy == OrderByDesc {
		sort.Slice(data, func(i, j int) bool { return data[i].Name > data[j].Name })
	}
	return data
}
func (req *SearchRequest) sortAge(data []User) []User {
	if req.OrderBy == OrderByAsc {
		sort.Slice(data, func(i, j int) bool { return data[i].Age < data[j].Age })
	} else if req.OrderBy == OrderByDesc {
		sort.Slice(data, func(i, j int) bool { return data[i].Age > data[j].Age })
	}
	return data
}

func setXmlDecoder(buf []byte) *xml.Decoder {
	input := bytes.NewReader(buf)
	decoder := xml.NewDecoder(input)
	return decoder
}
func setUsers(buf []byte) ([]User, error) {
	decoder := setXmlDecoder(buf)
	users := make([]User, 0, 70)
	var usr User
	nameParts := make([]string, 2)
	switcher := map[string]any{
		idTg:    &usr.Id,
		ageTg:   &usr.Age,
		aboutTg: &usr.About,
		gndrTg:  &usr.Gender,
	}

	for {
		tok, tokenErr := decoder.Token()
		if tokenErr != nil && tokenErr != io.EOF {
			return nil, tokenErr
		} else if tokenErr == io.EOF {
			return users, nil
		}
		if tok == nil {
			fmt.Println("t is nil break")
		}

		switch tok := tok.(type) {
		case xml.StartElement:
			elName := tok.Name.Local
			switch elName {
			case idTg, ageTg, aboutTg, gndrTg:
				if err := decoder.DecodeElement(switcher[elName], &tok); err != nil {
					return nil, err
				}
			case fNameTg:
				var s string
				if err := decoder.DecodeElement(&s, &tok); err != nil {
					return nil, err
				}
				nameParts[0] = s
			case lNameTg:
				var s string
				if err := decoder.DecodeElement(&s, &tok); err != nil {
					return nil, err
				}
				nameParts[1] = s
			}
		case xml.EndElement:
			elName := tok.Name.Local
			if elName == usrTg {
				usr.Name = strings.Join(nameParts, " ")
				users = append(users, usr)
				usr.reset()
				nameParts[0] = ""
				nameParts[1] = ""
			}
		}
	}

}

func (u *User) reset() {
	u.Id = 0
	u.Name = ""
	u.Age = 0
	u.About = ""
	u.Gender = ""
}

func parseRequest(r *http.Request) (*SearchRequest, error) {
	var srq SearchRequest
	vals := r.URL.Query()
	for _, f := range fields {
		val := vals.Get(f)

		if val == "" {
			continue
		}
		switch f {
		case limitF:
			v, err := strconv.Atoi(val)
			if err != nil {
				return nil, err
			}
			srq.Limit = v
		case offsetF:
			v, err := strconv.Atoi(val)
			if err != nil {
				return nil, err
			}
			srq.Offset = v
		case order_byF:
			v, err := strconv.Atoi(val)
			if err != nil {
				return nil, err
			}
			if checkOrder(v) {
				srq.OrderBy = v
			} else {
				return nil, errors.New(ErrorBadOrderField)
			}
		case queryF:
			srq.Query = strings.ToLower(val)
		case order_fieldF:
			val = strings.TrimSpace(strings.ToLower(val))
			if orderFieldIsAllowed(val) {
				srq.OrderField = val
			} else {
				return nil, errors.New(ErrorBadOrderField)
			}

		}

	}

	return &srq, nil

}

func checkOrder(i int) bool {
	if OrderByDesc != i && OrderByAsIs != i && OrderByAsc != i {
		return false
	}
	return true

}

func orderFieldIsAllowed(s string) bool {

	for _, elem := range allowedOF {
		if s == elem {
			return true
		}
	}
	return false

}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	xmlFile, err := os.ReadFile("dataset.xml")
	if err != nil {
		fmt.Println(err)
	}

	users, err := setUsers(xmlFile)
	if err != nil {
		writeResponse(http.StatusInternalServerError, errData(err), w)
	}

	sq, err := parseRequest(r)
	if err != nil {
		writeResponse(http.StatusInternalServerError, errData(err), w)
	} else {
		run(sq, users, w)
	}

}

func run(sq *SearchRequest, usrs []User, w http.ResponseWriter) {
	usrs = sq.search(usrs)
	usrs = sq.sort(usrs)
	data, err := json.Marshal(usrs)
	if err != nil {
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}
	writeResponse(http.StatusOK, data, w)
}

func errData(err error) []byte {
	resp := make(map[string]string)
	resp["Error"] = err.Error()
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}
	return jsonResp
}

func writeResponse(statusCode int, jsonData []byte, w http.ResponseWriter) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write(jsonData)
	if err != nil {
		writeResponse(http.StatusInternalServerError, nil, w)
		log.Fatalf("Error happened in http.ResponseWriter write. Err: %s", err)
	}
}

func TestSetUsers(t *testing.T) {
	var xmlData = []byte(`<?xml version="1.0" encoding="UTF-8" ?>
<root>
  <row>
    <id>0</id>
    <guid>1a6fa827-62f1-45f6-b579-aaead2b47169</guid>
    <isActive>false</isActive>
    <balance>$2,144.93</balance>
    <picture>http://placehold.it/32x32</picture>
    <age>22</age>
    <eyeColor>green</eyeColor>
    <first_name>Boyd</first_name>
    <last_name>Wolf</last_name>
    <gender>male</gender>
    <company>HOPELI</company>
    <email>boydwolf@hopeli.com</email>
    <phone>+1 (956) 593-2402</phone>
    <address>586 Winthrop Street, Edneyville, Mississippi, 9555</address>
    <about>Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.
</about>
    <registered>2017-02-05T06:23:27 -03:00</registered>
    <favoriteFruit>apple</favoriteFruit>
  </row>
  <row>
    <id>1</id>
    <guid>46c06b5e-dd08-4e26-bf85-b15d280e5e07</guid>
    <isActive>false</isActive>
    <balance>$2,705.71</balance>
    <picture>http://placehold.it/32x32</picture>
    <age>21</age>
    <eyeColor>green</eyeColor>
    <first_name>Hilda</first_name>
    <last_name>Mayer</last_name>
    <gender>female</gender>
    <company>QUINTITY</company>
    <email>hildamayer@quintity.com</email>
    <phone>+1 (932) 421-2117</phone>
    <address>311 Friel Place, Loyalhanna, Kansas, 6845</address>
    <about>Sit commodo consectetur minim amet ex. Elit aute mollit fugiat labore sint ipsum dolor cupidatat qui reprehenderit. Eu nisi in exercitation culpa sint aliqua nulla nulla proident eu. Nisi reprehenderit anim cupidatat dolor incididunt laboris mollit magna commodo ex. Cupidatat sit id aliqua amet nisi et voluptate voluptate commodo ex eiusmod et nulla velit.
</about>
    <registered>2016-11-20T04:40:07 -03:00</registered>
    <favoriteFruit>banana</favoriteFruit>
  </row>
</root>`)
	rightAnsw := []User{
		{
			0, "Boyd Wolf", 22,
			"Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n",
			"male",
		},
		{
			1, "Hilda Mayer", 21,
			"Sit commodo consectetur minim amet ex. Elit aute mollit fugiat labore sint ipsum dolor cupidatat qui reprehenderit. Eu nisi in exercitation culpa sint aliqua nulla nulla proident eu. Nisi reprehenderit anim cupidatat dolor incididunt laboris mollit magna commodo ex. Cupidatat sit id aliqua amet nisi et voluptate voluptate commodo ex eiusmod et nulla velit.\n",
			"female",
		},
	}
	result, err := setUsers(xmlData)
	if err != nil {
		t.Errorf("unexpected error: %#v", err)
	}
	if !reflect.DeepEqual(result, rightAnsw) {
		t.Errorf("wrong result, expected %#v, got %#v", rightAnsw, result)
	}
}

func TestParseRequest(t *testing.T) {
	SearchRequestParams := setSearchReqParams("25", "1", "Hilda", "name", strconv.Itoa(OrderByAsc))

	SearchRequestReq, err := http.NewRequest(http.MethodGet, "?"+SearchRequestParams.Encode(), nil)
	if err != nil {
		t.Errorf("unexpected error: %#v", err)
	}

	res, err := parseRequest(SearchRequestReq)
	if err != nil {
		t.Errorf("unexpected error: %#v", err)
	}

	right := &SearchRequest{25, 1, "hilda", "name", -1}
	if !reflect.DeepEqual(right, res) {
		t.Errorf(" wrong result, expected %#v, got %#v", right, res)
	}

}

func setSearchReqParams(limit, offset, query, order_field, order_by string) url.Values {
	SearchRequestParams := url.Values{}
	SearchRequestParams.Add("limit", limit)
	SearchRequestParams.Add("offset", offset)
	SearchRequestParams.Add("query", query)
	SearchRequestParams.Add("order_field", order_field)
	SearchRequestParams.Add("order_by", order_by)
	return SearchRequestParams
}

func TestSearchSrvSimple(t *testing.T) {
	srv := SearchClient{}
	req := SearchRequest{-1, -1, "", "", 0}
	nilRes, err := srv.FindUsers(req)
	var rightRes *SearchResponse = nil
	rightErr1 := errors.New("limit must be > 0")
	if !(reflect.DeepEqual(nilRes, rightRes) && reflect.DeepEqual(err, rightErr1)) {
		t.Errorf(" wrong result, expected [%#v,%#v], got [%#v,%#v]", rightRes, rightErr1, nilRes, err)
	}
	req.Limit = 0
	rightErr2 := errors.New("offset must be > 0")
	nilRes, err = srv.FindUsers(req)
	if !(reflect.DeepEqual(nilRes, rightRes) && reflect.DeepEqual(err, rightErr2)) {
		t.Errorf(" wrong result, expected [%#v,%#v], got [%#v,%#v]", rightRes, rightErr2, nilRes, err)
	}
}

func dummyFunc(w http.ResponseWriter, r *http.Request) {
	writeResponse(http.StatusOK, []byte{'0'}, w)
}

func TestSearchClientJson(t *testing.T) {
	req := SearchRequest{25, 1, "Hilda", "name", OrderByAsc}

	ts := httptest.NewServer(http.HandlerFunc(dummyFunc))
	srv := SearchClient{"", ts.URL}
	nilRes, errRes := srv.FindUsers(req)

	var rightRes *SearchResponse = nil
	rightErr := errors.New("cant unpack result json: json: cannot unmarshal number into Go value of type []main.User")

	if !(reflect.DeepEqual(nilRes, rightRes) && reflect.DeepEqual(errRes, rightErr)) {
		t.Errorf(" wrong result, expected [%#v,%#v], got [%#v,%#v]", rightRes, rightErr, nilRes, errRes)
	}
	ts.Close()
}

func TestSearchSrvUnknownErr(t *testing.T) {
	req := SearchRequest{25, 1, "Hilda", "name", OrderByAsc}
	srv := SearchClient{}
	ts := httptest.NewServer(http.HandlerFunc(dummyFunc))
	nilRes, errRes := srv.FindUsers(req)

	var rightRes *SearchResponse = nil
	rightErr := errors.New("unknown error Get \"?limit=26&offset=1&order_by=-1&order_field=name&query=Hilda\": unsupported protocol scheme \"\"")

	if !(reflect.DeepEqual(nilRes, rightRes) && reflect.DeepEqual(errRes, rightErr)) {
		t.Errorf(" wrong result, expected [%#v,%#v], got [%#v,%#v]", rightRes, rightErr, nilRes, errRes)
	}
	ts.Close()

}

func dummyStatusCheck(w http.ResponseWriter, r *http.Request) {
	s := r.FormValue("query")
	status, _ := strconv.Atoi(s)
	switch status {
	case http.StatusUnauthorized:
		writeResponse(http.StatusUnauthorized, []byte{}, w)
	case http.StatusInternalServerError:
		writeResponse(http.StatusInternalServerError, []byte{}, w)
	}
}

type httpTestSimpleErr struct {
	Status int
	Error  string
}

func TestSearchClientStatusErr(t *testing.T) {
	cases := []httpTestSimpleErr{
		{http.StatusUnauthorized, "Bad AccessToken"},
		{http.StatusInternalServerError, "SearchServer fatal error"},
	}

	ts := httptest.NewServer(http.HandlerFunc(dummyStatusCheck))
	srv := SearchClient{"", ts.URL}
	for i, c := range cases {
		str := strconv.Itoa(c.Status)
		req := SearchRequest{0, 0, str, "", 0}
		res, err := srv.FindUsers(req)

		var resRight *SearchResponse = nil
		errRight := errors.New(c.Error)

		if !reflect.DeepEqual(res, resRight) || !reflect.DeepEqual(err, errRight) {
			t.Errorf("%v case : wrong result, expected [%#v,%#v], got [%#v,%#v]", i, resRight, errRight, res, err)
		} else {
			t.Logf("%v case : good result, expected [%#v,%#v], got [%#v,%#v]", i, resRight, errRight, res, err)
		}
	}

	ts.Close()
}

var (
	badJsonErrCode       = 0
	badOrderFieldErrCode = 1
	badRequestErrCode    = 2
	ErrAnother           = "another error"
)

func dummyBadRequest(w http.ResponseWriter, r *http.Request) {
	c := r.FormValue("query")
	caseNum, _ := strconv.Atoi(c)
	switch caseNum {
	case badJsonErrCode:
		writeResponse(http.StatusBadRequest, []byte{'0'}, w)
	case badOrderFieldErrCode:
		sErrResp := SearchErrorResponse{ErrorBadOrderField}
		jsErr, _ := json.Marshal(sErrResp)
		writeResponse(http.StatusBadRequest, jsErr, w)
	case badRequestErrCode:
		sErrResp := SearchErrorResponse{ErrAnother}
		jsErr, _ := json.Marshal(sErrResp)
		writeResponse(http.StatusBadRequest, jsErr, w)
	}
}

func TestSearchClientBadRequest(t *testing.T) {
	cases := []string{
		"cant unpack error json: json: cannot unmarshal number into Go value of type main.SearchErrorResponse",
		"OrderField  invalid",
		"unknown bad request error: another error",
	}

	ts := httptest.NewServer(http.HandlerFunc(dummyBadRequest))
	srv := SearchClient{"", ts.URL}
	for i, c := range cases {
		req := SearchRequest{0, 0, strconv.Itoa(i), "", 0}
		res, err := srv.FindUsers(req)

		var resRight *SearchResponse = nil
		errRight := errors.New(c)

		if !reflect.DeepEqual(res, resRight) || !reflect.DeepEqual(err, errRight) {
			t.Errorf("%v case : wrong result, expected [%#v,%#v], got [%#v,%#v]", i, resRight, errRight, res, err)
		}
	}

	ts.Close()
}

func TestSearchClientGood(t *testing.T) {
	req := SearchRequest{
		26, 1, "Hilda", "name", OrderByAsc,
	}
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	srv := SearchClient{"", ts.URL}

	res, err := srv.FindUsers(req)

	var resRight = &SearchResponse{
		[]User{
			{1, "Hilda Mayer", 21,
				"Sit commodo consectetur minim amet ex. " +
					"Elit aute mollit fugiat labore sint ipsum dolor" +
					" cupidatat qui reprehenderit. Eu nisi in exercitation" +
					" culpa sint aliqua nulla nulla proident eu." +
					" Nisi reprehenderit anim cupidatat dolor incididunt " +
					"laboris mollit magna commodo ex. Cupidatat sit id aliqua " +
					"amet nisi et voluptate voluptate commodo ex eiusmod et nulla velit.\n",
				"female"},
		}, false,
	}
	var errRight error = nil

	if !reflect.DeepEqual(res, resRight) || !reflect.DeepEqual(err, errRight) {
		t.Errorf("wrong result, expected [%#v,%#v]", resRight, errRight)
		t.Errorf("got [%#v,%#v]", res, err)
	}

	ts.Close()
}
func TestSearchClientGoodLimit(t *testing.T) {
	req := SearchRequest{
		0, 1, "Hilda", "name", OrderByAsc,
	}
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	srv := SearchClient{"", ts.URL}

	res, err := srv.FindUsers(req)

	var resRight = &SearchResponse{
		[]User{},
		true}

	var errRight error = nil

	if !reflect.DeepEqual(res, resRight) || !reflect.DeepEqual(err, errRight) {
		t.Errorf("wrong result, expected [%#v,%#v]", resRight, errRight)
		t.Errorf("got [%#v,%#v]", res, err)
	}

	ts.Close()
}

func dummyTimeout(w http.ResponseWriter, r *http.Request) {
	time.Sleep(20 * time.Millisecond)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})

}

func TestSearchClientTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(dummyTimeout))

	http.DefaultTransport.(*http.Transport).ResponseHeaderTimeout = 10 * time.Millisecond

	srv := SearchClient{"", ts.URL}

	req := SearchRequest{0, 0, "", "", 0}
	res, err := srv.FindUsers(req)

	var resRight *SearchResponse = nil
	var errRight = errors.New("timeout for limit=1&offset=0&order_by=0&order_field=&query=")

	if !reflect.DeepEqual(res, resRight) || !reflect.DeepEqual(err, errRight) {
		t.Errorf("wrong result, expected [%#v,%#v], got [%#v,%#v]", resRight, errRight, res, err)
	}

	ts.Close()
}
