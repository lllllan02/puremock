package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

func mock(path string) error {

	// 获取当前目录的绝对路径
	dir, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// 获取GOPATH的值
	gopath := os.Getenv("GOPATH")
	fmt.Printf("gopath: %v\n", gopath)

	// 获取包的相对路径
	relPath, err := filepath.Rel(filepath.Join(gopath, "src"), dir)
	if err != nil {
		return err
	}

	// 打印包的完整导入路径
	fmt.Printf("relPath: %v\n", relPath)

	// Parse the Go file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return err
	}

	pkg := file.Name.Name
	fmt.Printf("pkg: %v\n", pkg)

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		fmt.Printf("genDecl: %+v\n", genDecl)
		// genDecl.Tok: import, type

		for _, spec := range genDecl.Specs {
			if importSpec, ok := spec.(*ast.ImportSpec); ok {
				importName := importSpec.Name
				importPath := importSpec.Path.Value
				fmt.Printf("	importName: %v\n", importName)
				fmt.Printf("	importPath: %v\n", importPath)
			}

			if typeSpec, ok := spec.(*ast.TypeSpec); ok {
				fmt.Printf("	typeSpec: %+v\n", typeSpec)

				interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
				if !ok {
					continue
				}

				fmt.Printf("	interfaceType: %+v\n", interfaceType)

				for _, method := range interfaceType.Methods.List {
					ft, ok := method.Type.(*ast.FuncType)
					if !ok {
						continue
					}

					fmt.Printf("funcType(ft, method.Names...): %v\n", funcType(ft, method.Names...))
				}
			}

		}
	}

	fmt.Printf("file.Imports: %+v\n", file.Imports)

	return nil
}

func paramType(param ast.Expr) string {
	if param == nil {
		return ""
	}

	var t string
	switch pt := param.(type) {
	case *ast.Ident:
		t = pt.Name
	case *ast.Ellipsis:
		t = fmt.Sprintf("...%s", paramType(pt.Elt))
	case *ast.BasicLit:
		t = pt.Value
	case *ast.ArrayType:
		t = fmt.Sprintf("[%s]%s", paramType(pt.Len), paramType(pt.Elt))
	case *ast.MapType:
		t = fmt.Sprintf("map[%s]%s", paramType(pt.Key), paramType(pt.Value))
	case *ast.SelectorExpr:
		// TODO 如果 mock 地址与原地址不在同一个包，并且使用了源包中的类型，需要引用源包
		t = fmt.Sprintf("%s.%s", pt.X, pt.Sel)
	case *ast.StarExpr:
		t = fmt.Sprintf("*%s", paramType(pt.X))
	case *ast.InterfaceType:
		t = "interface"
	case *ast.ChanType:
		t = fmt.Sprintf("chan %s", paramType(pt.Value))
	case *ast.FuncType:
		t = funcType(pt)
	default:
		t = fmt.Sprint(reflect.TypeOf(param))
	}

	return t
}

func funcType(ft *ast.FuncType, name ...*ast.Ident) string {
	var f string = "func"
	if len(name) > 0 {
		f = fmt.Sprintf("%s %s", f, name[0].Name)
	}

	var params []string
	if ft.Params != nil {
		for _, param := range ft.Params.List {
			params = append(params, fmt.Sprintf("%s%s", names(param.Names...), paramType(param.Type)))
		}
	}
	f = fmt.Sprintf("%s(%s)", f, strings.Join(params, ", "))

	var results []string
	if ft.Results != nil {
		for _, result := range ft.Results.List {
			results = append(results, fmt.Sprintf("%s%s", names(result.Names...), paramType(result.Type)))
		}
	}
	if n := ft.Results.NumFields(); n == 1 {
		f = fmt.Sprintf("%s %s", f, results[0])
	} else if n > 1 {
		f = fmt.Sprintf("%s (%s)", f, strings.Join(results, ", "))
	}

	return f
}

func names(name ...*ast.Ident) string {
	if l := len(name); l == 0 {
		return ""
	} else if l == 1 {
		return name[0].Name
	}

	var s []string
	for _, n := range name {
		s = append(s, n.Name)
	}
	return strings.Join(s, ", ") + " "
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
