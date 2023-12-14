package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

var (
	source      *string
	destination *string
	packageName *string
	disjoint    bool
	importName  string

	importMap map[string]bool
	typeMap   map[string]bool
)

func main() {
	source = flag.String("source", "", "(source mode) Input Go source file; enables source mode.")
	destination = flag.String("destination", "", "Output file; defaults to stdout.")
	packageName = flag.String("package", "", "Package of the generated code; defaults to the package of the input")

	flag.Parse()

	disjoint = filepath.Dir(*source) != filepath.Dir(*destination)
	if *source == "" {
		fmt.Println("You must set the source.")
		flag.PrintDefaults()
		return
	}

	content, err := mock(*source)
	if err != nil {
		panic(err)
	}

	if *destination != "" {
		file, err := os.Create(*destination)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		if _, err = file.WriteString(content); err != nil {
			panic(err)
		}
	} else {
		fmt.Println(content)
	}
}

type Mock struct {
	Name  string
	Funcs []string
}

func (m Mock) String() string {
	if len(m.Funcs) == 0 {
		return fmt.Sprintf("type Mock%s struct{}", m.Name)
	}
	return fmt.Sprintf("type %s struct{}\n\n%s", m.Name, strings.Join(m.Funcs, "\n\n"))
}

func mock(path string) (string, error) {
	var (
		content = ""
		pkgName = ""
		imports = []string{}
		mocks   = []Mock{}
	)
	importMap = make(map[string]bool)
	typeMap = make(map[string]bool)

	// Parse the Go file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return "", err
	}
	pkgName = file.Name.Name
	if packageName != nil && *packageName != "" {
		pkgName = *packageName
	}
	importName = genImportName(file.Name.Name, file.Imports)

	// type ... interface{}
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			typeMap[typeSpec.Name.Name] = true

			interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}

			mock := Mock{Name: fmt.Sprintf("Mock%s", typeSpec.Name.Name)}
			for _, method := range interfaceType.Methods.List {
				ft, ok := method.Type.(*ast.FuncType)
				if !ok {
					continue
				}

				if ft.Results == nil {
					mock.Funcs = append(mock.Funcs, fmt.Sprintf("func (%s) %s {\n}", mock.Name, funcType(ft, method.Names, true)))
				} else {
					mock.Funcs = append(mock.Funcs, fmt.Sprintf("func (%s) %s {\n    return\n}", mock.Name, funcType(ft, method.Names, true)))
				}

			}
			mocks = append(mocks, mock)
		}
	}

	content = fmt.Sprintf("package %s", pkgName)
	for _, ipt := range file.Imports {
		path := strings.Trim(ipt.Path.Value, "\"")
		name := filepath.Base(path)
		if ipt.Name != nil {
			name = ipt.Name.Name
		}
		if importMap[name] {
			imports = append(imports, fmt.Sprintf("    %s \"%s\"", name, path))
		}
	}
	if importMap[importName] {
		path, err := GetPackagePath(*source)
		if err != nil {
			panic(err)
		}
		imports = append(imports, fmt.Sprintf("    %s \"%s\"", importName, path))
	}
	if len(imports) > 0 {
		content = fmt.Sprintf("%s\n\nimport(\n%s\n)", content, strings.Join(imports, "\n"))
	}
	for _, mock := range mocks {
		content = fmt.Sprintf("%s\n\n%s", content, mock.String())
	}

	return content, nil
}

func funcType(ft *ast.FuncType, name []*ast.Ident, resultNumber bool) (fun string) {
	var f string
	if len(name) > 0 {
		f = name[0].Name
	}

	var params []string
	if ft.Params != nil {
		params = names(ft.Params.List, false)
	}
	f = fmt.Sprintf("%s(%s)", f, strings.Join(params, ", "))

	var results []string
	if ft.Results != nil {
		results = names(ft.Results.List, resultNumber)
	}
	if n := len(results); resultNumber && n > 0 {
		f = fmt.Sprintf("%s (%s)", f, strings.Join(results, ", "))
	} else if !resultNumber && n == 1 {
		f = fmt.Sprintf("%s %s", f, results[0])
	} else if !resultNumber && n > 1 {
		f = fmt.Sprintf("%s (%s)", f, strings.Join(results, ", "))
	}

	return f
}

func names(fields []*ast.Field, number bool) []string {
	var index int = 0

	var params []string
	for _, field := range fields {
		var count int = 1
		if n := len(field.Names); n > 0 {
			count = n
		}
		pt := paramType(field.Type)

		if number {
			var ns []string
			for i := 0; i < count; i++ {
				ns = append(ns, fmt.Sprintf("arg%d", index))
				index++
			}
			params = append(params, fmt.Sprintf("%s %s", strings.Join(ns, ", "), pt))
		} else {
			for i := 0; i < count; i++ {
				params = append(params, pt)
			}
		}
	}

	return params
}

func paramType(param ast.Expr) string {
	if param == nil {
		return ""
	}

	var t string
	switch pt := param.(type) {
	case *ast.Ident:
		// 如果 mock 地址与原地址不在同一个包，并且使用了源包中的类型，需要引用源包
		if disjoint && typeMap[pt.Name] {
			importMap[importName] = true
			t = fmt.Sprintf("%s.%s", importName, pt.Name)
		} else {
			t = pt.Name
		}
	case *ast.Ellipsis:
		t = fmt.Sprintf("...%s", paramType(pt.Elt))
	case *ast.BasicLit:
		t = pt.Value
	case *ast.ArrayType:
		t = fmt.Sprintf("[%s]%s", paramType(pt.Len), paramType(pt.Elt))
	case *ast.MapType:
		t = fmt.Sprintf("map[%s]%s", paramType(pt.Key), paramType(pt.Value))
	case *ast.SelectorExpr:
		importMap[paramType(pt.X)] = true
		t = fmt.Sprintf("%s.%s", pt.X, pt.Sel)
	case *ast.StarExpr:
		t = fmt.Sprintf("*%s", paramType(pt.X))
	case *ast.InterfaceType:
		t = "interface{}"
	case *ast.ChanType:
		t = fmt.Sprintf("chan %s", paramType(pt.Value))
	case *ast.FuncType:
		t = fmt.Sprintf("func %s", funcType(pt, nil, false))
	default:
		t = fmt.Sprint(reflect.TypeOf(param))
	}

	return t
}

func genImportName(name string, imports []*ast.ImportSpec) string {
	m := make(map[string]bool)
	for _, ipt := range imports {
		path := strings.Trim(ipt.Path.Value, "\"")
		name := filepath.Base(path)
		if ipt.Name != nil {
			name = ipt.Name.Name
		}
		m[name] = true
	}
	if !m[name] {
		return name
	}
	for i := 0; ; i++ {
		n := fmt.Sprintf("%s%d", name, i)
		if !m[n] {
			return n
		}
	}
}

func GetPackagePath(path string) (pkg string, err error) {
	// 获取目标文件的绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("get absolute path: %w", err)
	}

	// 获取目标文件所在的目录
	absDir := filepath.Dir(absPath)

	// 查找go.mod文件
	var data []byte
	var dir, pre, gomod = absDir, "", ""
	for {
		gomod = filepath.Join(dir, "go.mod")
		if data, err = os.ReadFile(gomod); err == nil {
			break
		}

		if dir, pre = filepath.Dir(dir), dir; dir == pre {
			return "", fmt.Errorf("no go.mod found")
		}
	}

	// 获取模块名
	modLine := strings.Split(string(data), "\n")[0]
	parts := strings.Split(modLine, " ")
	if len(parts) != 2 || parts[0] != "module" {
		return "", fmt.Errorf("invalid go.mod: %s", modLine)
	}
	modName := parts[1]

	// 获取包相对于模块根目录的路径
	relPath, err := filepath.Rel(dir, absDir)
	if err != nil {
		return "", fmt.Errorf("get relative path: %w", err)
	}

	pkg = filepath.Join(modName, relPath)
	return
}
