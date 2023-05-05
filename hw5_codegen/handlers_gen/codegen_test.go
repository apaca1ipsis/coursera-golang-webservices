package main

import (
	"net/http"
	"reflect"
	"testing"
)

type ValidatorTestCase struct {
	caseStr  string
	expected *tagParams
}
type ValidatorTestCaseWithName struct {
	ValidatorTestCase
	fieldName string
}

// go test ./handlers_gen
const (
	intType    = "int"
	stringType = "string"
)

func TestSetValidator(t *testing.T) {
	n10 := 10
	n0 := 0
	n128 := 128
	cases := []ValidatorTestCase{
		{`apivalidator:"required"`,
			&tagParams{"", "", true,
				[]string(nil), nil, nil, nil},
		},
		{`apivalidator:"required,min=10"`,
			&tagParams{"", "", true,
				[]string(nil), nil, &n10,
				nil},
		},
		{`apivalidator:"paramname=full_name"`,
			&tagParams{"", "full_name", false,
				[]string(nil), nil, nil, nil},
		},
		{`apivalidator:"enum=user|moderator|admin,default=user"`,
			&tagParams{"", "", false,
				[]string{"user", "moderator", "admin"}, "user",
				nil, nil,
			}},
		{`apivalidator:"min=0,max=128"`,
			&tagParams{"", "", false,
				[]string(nil), nil, &n0,
				&n128,
			},
		},
	}

	for i, c := range cases {
		result, err := setTgParams("", c.caseStr)
		if err != nil {
			t.Errorf("[%d] error %#v", i, err)
			continue
		}

		if !reflect.DeepEqual(result, c.expected) {
			t.Errorf("[%d] results not match\nGot: %#v\nExpected: %#v", i, result, c.expected)
			continue
		} else {
			t.Logf("[%d] results match!", i)
		}
	}
}

type ApigenTestCase struct {
	s      string
	result *methodTagParams
}

const (
	ApiUserCreate  = "/user/create"
	ApiUserProfile = "/user/profile"
)

func TestParseApigenMark(t *testing.T) {
	methodPost := http.MethodPost
	cases := []ApigenTestCase{
		{`apigen:api {"Url": "/user/profile", "auth": false}`,
			&methodTagParams{ApiUserProfile, false, nil},
		},
		{`apigen:api {"Url": "/user/create", "auth": true, "method": "POST"}`,
			&methodTagParams{ApiUserCreate, true, &methodPost},
		},
	}

	for i, cs := range cases {
		expected, err := parseApigenMark(cs.s)
		if err != nil {
			t.Logf("[%d] unexpected error: %v", i, err)
			continue
		}
		if !reflect.DeepEqual(expected, cs.result) {
			t.Errorf("[%d] results not match\nGot: %#v\nExpected: %#v", i, expected, cs.result)
			continue
		} else {
			t.Logf("[%d] results match!", i)
		}
	}
}
