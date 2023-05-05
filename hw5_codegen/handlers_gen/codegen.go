package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"unicode"
)

// код писать тут
// codegen

type tplStruct struct {
	StructName string
	Params     []tplParam
}

type tplParam struct {
	ParamType string
	tagParams
}

type tagParams struct {
	Name         string
	ParamName    string
	IsRequired   bool
	Enum         []string
	DefaultValue any
	Min          *int
	Max          *int
}

type fieldParams struct {
	FName string
	FTag  string
	FType string
}

func setTplParam(field fieldParams) (*tplParam, error) {
	var t tplParam
	tTg, err := setTgParams(field.FName, field.FTag)
	if err != nil {
		return nil, err
	}
	t.tagParams = *tTg
	t.ParamType = field.FType
	return &t, nil
}

func setTgParams(fieldName, s string) (*tagParams, error) {
	var t tagParams
	if strings.Contains(s, "required") {
		t.IsRequired = true
	}
	params := strings.Split(s, ",")
	for _, p := range params {
		if strings.Contains(p, "paramname=") {
			arrP := strings.Split(p, "=")
			t.ParamName = strings.Trim(arrP[1], "\"")
		}

		if strings.Contains(p, "min=") {
			arrP := strings.Split(p, "=")
			minVal, err := strconv.Atoi(strings.Trim(arrP[1], "\""))
			if err != nil {
				return nil, err
			}
			t.Min = &minVal
		}

		if strings.Contains(p, "max=") {
			arrP := strings.Split(p, "=")
			maxVal, err := strconv.Atoi(strings.Trim(arrP[1], "\""))
			if err != nil {
				return nil, err
			}
			t.Max = &maxVal
		}

		if strings.Contains(p, "default=") {
			arrP := strings.Split(p, "=")
			t.DefaultValue = strings.Trim(arrP[1], "\"")
		}

		if strings.Contains(p, "enum=") {
			arrP := strings.Split(p, "=")
			enum := strings.Split(strings.Trim(arrP[1], "\""), "|")
			t.Enum = enum
		}
	}

	if t.ParamName == "" {
		t.ParamName = strings.ToLower(fieldName)
	}
	t.Name = fieldName
	return &t, nil
}
func myStrTitleFoo(s string) string {
	if len(s) == 0 {
		return s
	}
	var res []rune
	for ind, val := range s {
		if ind == 0 && unicode.IsLetter(val) { //check if character is a letter convert the first character to upper case
			res = append(res, unicode.ToUpper(val))
		} else {
			res = append(res, val)
		}
	}
	return string(res)
}

type Apigen struct {
	Structs  map[string]tplStruct
	Handlers map[string][]*tplMethod
	out      *os.File
}

func (a *Apigen) addMethod(m *tplMethod) {
	a.Handlers[m.ReceiverType] = append(a.Handlers[m.ReceiverType], m)
}

func (a *Apigen) addStruct(tplS tplStruct) error {
	_, ok := a.Structs[tplS.StructName]
	if ok {
		msg := fmt.Sprintf("structure %v was redeclared, check api file", tplS.StructName)
		return errors.New(msg)
	} else {
		a.Structs[tplS.StructName] = tplS
		return nil
	}
}

func (a *Apigen) init() {
	// Раскомментить
	var err error
	//out, _ := os.Create(os.Args[2])
	a.out, err = os.Create("vikofile.go")
	if err != nil {
		log.Fatal(err)
	}
	a.Structs = make(map[string]tplStruct)
	a.Handlers = make(map[string][]*tplMethod)
}

func main() {
	fset := token.NewFileSet()
	// Раскомментить
	// node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	node, err := parser.ParseFile(fset, "api.go", nil, parser.ParseComments)

	if err != nil {
		log.Fatal(err)
	}

	var codegen Apigen
	codegen.init()

	// это пишет в out-файл код
	fmt.Fprintln(codegen.out, `package `+node.Name.Name)
	fmt.Fprintln(codegen.out, `import (
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
)`)

	// разбор файла откуда
	for _, f := range node.Decls {
		fD, ok := f.(*ast.FuncDecl)
		if ok {
			fmt.Printf("%T is *ast.FuncDecl\n", fD)
			funcComment := fD.Doc.Text()
			if checkMethodMark(funcComment) {
				fmt.Printf("\thas %v mark\n", apigenMark)
				method, err := setTplMethod(fD)
				if err != nil {
					log.Fatal(err)
				}
				codegen.addMethod(method)
			} else {
				fmt.Printf("\tno mark\n")
			}
			continue
		}
		g, ok := f.(*ast.GenDecl)
		if !ok {
			fmt.Printf("SKIP %T is not *ast.GenDecl\n", f)
			continue
		}
		codegen.setStruct(g)
	}
	codegen.generateFile()
}

// apigen:api {"Url": "/user/create", "auth": true, "method": "POST"}
func parseApigenMark(s string) (*methodTagParams, error) {
	s = strings.ReplaceAll(s, apigenMark, "")
	s = strings.TrimSpace(s)
	js := []byte(s)
	var tmpl methodTagParams
	err := json.Unmarshal(js, &tmpl)
	if err != nil {
		return nil, err
	}
	return &tmpl, nil
}

const (
	handlerPrefix = "handler"
	apigenMark    = "apigen:api"
)

func checkMethodMark(s string) bool {
	if strings.HasPrefix(s, apigenMark) {
		return true
	}
	return false
}

func setMethodInfo(fD *ast.FuncDecl) (*methodInfo, error) {
	methodName := fD.Name.Name
	rcvInfo, ok := fD.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		msg := fmt.Sprintf("'can't parse receiver from %v", methodName)
		return nil, errors.New(msg)
	}

	prms := fD.Type.Params.List
	params, okP := parseFieldList(prms)
	if !okP {
		msg := fmt.Sprintf("'can't parse Params from %v", methodName)
		return nil, errors.New(msg)
	}

	rslts := fD.Type.Results.List // for elem, elem.Type.X.Name
	results, okR := parseFieldList(rslts)
	if !okR {
		msg := fmt.Sprintf("'can't parse results from %v", methodName)
		return nil, errors.New(msg)
	}

	rcvStructName := rcvInfo.X.(*ast.Ident).Name

	return &methodInfo{rcvStructName,
		methodName, params, results}, nil
}

func parseFieldList(ls []*ast.Field) ([]string, bool) {
	var result []string
	for _, elem := range ls {
		var pName string
		param, ok := elem.Type.(*ast.SelectorExpr)
		if !ok {
			var param *ast.Ident
			param, ok = elem.Type.(*ast.Ident)
			if !ok {
				var param *ast.StarExpr
				param, ok = elem.Type.(*ast.StarExpr)
				if !ok {
					return nil, false
				} else {
					pName = param.X.(*ast.Ident).Name
				}
			} else {
				pName = param.Name
			}
		} else {
			objName := param.X.(*ast.Ident).Name
			pkgName := param.Sel.Name
			pName = pkgName + "." + objName
		}
		if pName == "context" {
			b := 1
			_ = b
		}
		result = append(result, pName)
	}
	return result, true
}

func (codegen *Apigen) setStruct(g *ast.GenDecl) {

SPECS_LOOP:
	for _, spec := range g.Specs {
		currType, ok := spec.(*ast.TypeSpec)
		// все импорты скипает
		if !ok {
			fmt.Printf("SKIP %T is not ast.TypeSpec\n", spec)
			continue
		}

		fileStruct, ok := currType.Type.(*ast.StructType)
		if !ok {
			fmt.Printf("SKIP %T is not ast.StructType\n", fileStruct)
			continue
		}
		fmt.Printf("process struct %s\n", currType.Name.Name)
		fmt.Printf("\tgenerating set/validate method\n")

		curStruct := tplStruct{currType.Name.Name, nil}
		if len(fileStruct.Fields.List) == 0 {
			fmt.Printf("SKIP %#v doesnt have fields\n", currType.Name.Name)
			continue SPECS_LOOP
		}
		//FIELDS_LOOP:
		for _, field := range fileStruct.Fields.List {
			if field.Tag == nil {
				fmt.Printf("SKIP field %#v doesnt have apivalidator mark\n", currType.Name.Name)
				continue SPECS_LOOP
			}

			tag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])
			validatorStr := tag.Get("apivalidator")

			if validatorStr == "" {
				fmt.Printf("SKIP field %#v doesnt have apivalidator mark\n", currType.Name.Name)
				continue SPECS_LOOP
			}

			fName := field.Names[0].Name
			fType := field.Type.(*ast.Ident).Name

			params := fieldParams{fName, validatorStr, fType}
			var tpl *tplParam
			var err error
			tpl, err = setTplParam(params)
			if err != nil {
				log.Fatal(err)
			}
			curStruct.Params = append(curStruct.Params, *(tpl))
		}
		errS := codegen.addStruct(curStruct)
		if errS != nil {
			log.Fatal(errS)
		}
	}
}

func (codegen *Apigen) generateFile() {
	//-=================================
	// NOTABENE : прикольно, здесь я скомпилировала два шаблона, и потом последовательно их вызвала
	/*			innerT := template.Must(template.New("structCode").ParseFiles("handlers_gen/structCode.tmpl"))
				template.Must(innerT.New("fieldCode").ParseFiles("handlers_gen/fieldCode.tmpl"))
				err = innerT.ExecuteTemplate(out, "structCode", curStruct)
				err = innerT.ExecuteTemplate(out, "fieldCode", curStruct)*/
	//===================================
	// а тут я распарсила шаблоны в одну переменную, и вызвала внутренний шаблон внутри внешнего
	funcMap := template.FuncMap{
		"ToUpper": myStrTitleFoo,
	}
	innerT := template.Must(template.New("structCode.tmpl").Funcs(funcMap).ParseFiles(
		"handlers_gen/structCode.tmpl", "handlers_gen/fieldCode.tmpl"))

	serverT := template.Must(template.New("server.tmpl").Funcs(funcMap).ParseFiles(
		"handlers_gen/server.tmpl", "handlers_gen/handler.tmpl"))

	commonT := template.Must(template.New("commonFuncs.tmpl").Funcs(funcMap).ParseFiles(
		"handlers_gen/commonFuncs.tmpl"))

	for _, s := range codegen.Structs {
		err := innerT.Execute(codegen.out, s)
		if err != nil {
			log.Fatalf("Error: %v, in %v generating set/validate", err, s)
		}
	}

	for objName, objMethods := range codegen.Handlers {
		data := map[string][]*tplMethod{objName: objMethods}
		err := serverT.Execute(codegen.out, data)
		if err != nil {
			log.Fatalf("Error: %v, in %v generating ServeHTTP", err, objName)
		}
	}

	err := commonT.Execute(codegen.out, nil)
	if err != nil {
		log.Fatalf("Error %v in rendering common part", err)
	}

	fmt.Fprintln(codegen.out, "")
}

type tplMethod struct {
	methodTagParams
	methodInfo
}

type methodTagParams struct {
	Url            string  `json:"url"`
	NeedAuth       bool    `json:"auth"`
	AllowedRequest *string `json:"method"`
}

type methodInfo struct {
	ReceiverType string
	MethodName   string
	Params       []string
	ReturnValues []string
}

func (t *tplMethod) setTagAndInfo(mT methodTagParams, m methodInfo) {
	t.methodInfo = m
	t.methodTagParams = mT
}

func (t *tplMethod) parseApigenMark(s string) error {
	mTg, err := parseApigenMark(s)
	if err != nil {
		return err
	}
	t.methodTagParams = *mTg
	return nil
}

func (t *tplMethod) setMethodInfo(d *ast.FuncDecl) error {
	mI, err := setMethodInfo(d)
	if err != nil {
		return err
	}
	t.methodInfo = *mI
	return nil
}

func setTplMethod(fD *ast.FuncDecl) (*tplMethod, error) {
	funcComment := fD.Doc.Text()

	var method tplMethod
	erMark := method.parseApigenMark(funcComment)
	if erMark != nil {
		return nil, erMark
	}
	erMI := method.setMethodInfo(fD)
	if erMI != nil {
		return nil, erMI
	}
	return &method, nil

}
