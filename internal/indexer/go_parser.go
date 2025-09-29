package indexer

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"
)

// GoParser parses Go source code and extracts structural information
type GoParser struct {
	fileSet *token.FileSet
}

// ParsedCode represents the parsed structure of a Go file
type ParsedCode struct {
	PackageName  string            `json:"package_name"`
	Imports      []Import          `json:"imports"`
	Functions    []Function        `json:"functions"`
	Methods      []Method          `json:"methods"`
	Types        []TypeDef         `json:"types"`
	Constants    []Constant        `json:"constants"`
	Variables    []Variable        `json:"variables"`
	Interfaces   []Interface       `json:"interfaces"`
	Comments     []Comment         `json:"comments"`
	Dependencies []string          `json:"dependencies"`
	Complexity   CodeComplexity    `json:"complexity"`
	Metadata     map[string]string `json:"metadata"`
}

// Import represents an import statement
type Import struct {
	Name     string `json:"name"` // import alias
	Path     string `json:"path"` // import path
	Line     int    `json:"line"`
	Used     bool   `json:"used"`
	Standard bool   `json:"standard"` // whether it's a standard library import
}

// Function represents a function declaration
type Function struct {
	Name        string      `json:"name"`
	Signature   string      `json:"signature"`
	Parameters  []Parameter `json:"parameters"`
	Returns     []Return    `json:"returns"`
	StartLine   int         `json:"start_line"`
	EndLine     int         `json:"end_line"`
	Visibility  string      `json:"visibility"` // public, private
	Body        string      `json:"body"`
	DocString   string      `json:"doc_string"`
	Complexity  int         `json:"complexity"`
	IsTest      bool        `json:"is_test"`
	IsBenchmark bool        `json:"is_benchmark"`
	CallsTo     []string    `json:"calls_to"`
}

// Method represents a method declaration
type Method struct {
	Function              // Embed Function
	Receiver     Receiver `json:"receiver"`
	ReceiverType string   `json:"receiver_type"`
}

// Parameter represents a function parameter
type Parameter struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Return represents a return value
type Return struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type"`
}

// Receiver represents a method receiver
type Receiver struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	IsPointer bool   `json:"is_pointer"`
}

// TypeDef represents a type definition
type TypeDef struct {
	Name       string            `json:"name"`
	Kind       string            `json:"kind"` // struct, type, alias
	StartLine  int               `json:"start_line"`
	EndLine    int               `json:"end_line"`
	Fields     []Field           `json:"fields,omitempty"`
	Methods    []string          `json:"methods,omitempty"`
	DocString  string            `json:"doc_string"`
	Underlying string            `json:"underlying,omitempty"` // for type aliases
	Tags       map[string]string `json:"tags,omitempty"`
}

// Field represents a struct field
type Field struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Tag       string `json:"tag,omitempty"`
	DocString string `json:"doc_string,omitempty"`
	Line      int    `json:"line"`
	Embedded  bool   `json:"embedded"`
	Exported  bool   `json:"exported"`
}

// Interface represents an interface declaration
type Interface struct {
	Name      string            `json:"name"`
	StartLine int               `json:"start_line"`
	EndLine   int               `json:"end_line"`
	Methods   []InterfaceMethod `json:"methods"`
	DocString string            `json:"doc_string"`
	Embedded  []string          `json:"embedded,omitempty"` // embedded interfaces
}

// InterfaceMethod represents a method in an interface
type InterfaceMethod struct {
	Name       string      `json:"name"`
	Parameters []Parameter `json:"parameters"`
	Returns    []Return    `json:"returns"`
	DocString  string      `json:"doc_string"`
}

// Constant represents a constant declaration
type Constant struct {
	Name      string `json:"name"`
	Type      string `json:"type,omitempty"`
	Value     string `json:"value"`
	Line      int    `json:"line"`
	DocString string `json:"doc_string"`
}

// Variable represents a variable declaration
type Variable struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Value     string `json:"value,omitempty"`
	Line      int    `json:"line"`
	DocString string `json:"doc_string"`
	IsGlobal  bool   `json:"is_global"`
}

// Comment represents a comment block
type Comment struct {
	Text      string `json:"text"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Type      string `json:"type"` // line, block, doc
}

// CodeComplexity represents complexity metrics
type CodeComplexity struct {
	CyclomaticComplexity int     `json:"cyclomatic_complexity"`
	LinesOfCode          int     `json:"lines_of_code"`
	FunctionCount        int     `json:"function_count"`
	TypeCount            int     `json:"type_count"`
	InterfaceCount       int     `json:"interface_count"`
	TestCoverage         float64 `json:"test_coverage,omitempty"`
}

// NewGoParser creates a new Go parser
func NewGoParser() *GoParser {
	return &GoParser{
		fileSet: token.NewFileSet(),
	}
}

// ParseFile parses a Go source file and returns structured information
func (gp *GoParser) ParseFile(filename string, content string) (*ParsedCode, error) {
	// Parse the source code
	astFile, err := parser.ParseFile(gp.fileSet, filename, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file: %w", err)
	}

	parsed := &ParsedCode{
		PackageName:  astFile.Name.Name,
		Imports:      make([]Import, 0),
		Functions:    make([]Function, 0),
		Methods:      make([]Method, 0),
		Types:        make([]TypeDef, 0),
		Constants:    make([]Constant, 0),
		Variables:    make([]Variable, 0),
		Interfaces:   make([]Interface, 0),
		Comments:     make([]Comment, 0),
		Dependencies: make([]string, 0),
		Metadata:     make(map[string]string),
	}

	fmt.Printf("🔍 [DEBUG] Parsing %s with %d declarations\n", filename, len(astFile.Decls))

	// Parse top-level declarations with debug logging
	for i, decl := range astFile.Decls {
		fmt.Printf("  %d. Processing declaration type: %T\n", i+1, decl)

		switch d := decl.(type) {
		case *ast.GenDecl:
			switch d.Tok {
			case token.TYPE:
				fmt.Printf("    Processing TYPE declaration with %d specs\n", len(d.Specs))
				for j, spec := range d.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						fmt.Printf("      Spec %d: Type %s\n", j+1, typeSpec.Name.Name)

						typeDef := TypeDef{
							Name:      typeSpec.Name.Name,
							StartLine: gp.fileSet.Position(typeSpec.Pos()).Line,
							EndLine:   gp.fileSet.Position(typeSpec.End()).Line,
							DocString: gp.getDocString(d.Doc),
						}

						switch t := typeSpec.Type.(type) {
						case *ast.StructType:
							typeDef.Kind = "struct"
							fmt.Printf("        -> Struct type\n")
						case *ast.InterfaceType:
							typeDef.Kind = "interface"
							fmt.Printf("        -> Interface type\n")
							// Handle interface separately
							interfaceDef := Interface{
								Name:      typeSpec.Name.Name,
								StartLine: typeDef.StartLine,
								EndLine:   typeDef.EndLine,
								DocString: typeDef.DocString,
								Methods:   make([]InterfaceMethod, 0),
							}
							parsed.Interfaces = append(parsed.Interfaces, interfaceDef)
							continue
						case *ast.Ident:
							typeDef.Kind = "alias"
							typeDef.Underlying = t.Name
							fmt.Printf("        -> Type alias: %s\n", t.Name)
						default:
							typeDef.Kind = "type"
							typeDef.Underlying = "complex"
							fmt.Printf("        -> Other type: %T\n", t)
						}

						parsed.Types = append(parsed.Types, typeDef)
					}
				}
			case token.CONST:
				fmt.Printf("    Processing CONST declaration\n")
				// Handle constants...
			case token.VAR:
				fmt.Printf("    Processing VAR declaration\n")
				// Handle variables...
			}

		case *ast.FuncDecl:
			fmt.Printf("    Processing FUNCTION declaration\n")
			if d.Name != nil {
				fmt.Printf("      Function name: %s\n", d.Name.Name)

				function := Function{
					Name:        d.Name.Name,
					StartLine:   gp.fileSet.Position(d.Pos()).Line,
					EndLine:     gp.fileSet.Position(d.End()).Line,
					Visibility:  gp.getVisibility(d.Name.Name),
					DocString:   gp.getDocString(d.Doc),
					IsTest:      strings.HasPrefix(d.Name.Name, "Test"),
					IsBenchmark: strings.HasPrefix(d.Name.Name, "Benchmark"),
					CallsTo:     make([]string, 0),
					Complexity:  1, // Basic complexity
				}

				if d.Type != nil {
					function.Signature = gp.funcTypeToString(d.Type)
				}

				if d.Recv != nil {
					// It's a method
					fmt.Printf("        -> This is a method\n")
					method := Method{
						Function: function,
					}
					parsed.Methods = append(parsed.Methods, method)
				} else {
					// It's a function
					fmt.Printf("        -> This is a function\n")
					parsed.Functions = append(parsed.Functions, function)
				}
			} else {
				fmt.Printf("      WARNING: Function with nil name\n")
			}
		}
	}

	fmt.Printf("✅ [DEBUG] Parsed %s: %d functions, %d methods, %d types, %d interfaces\n",
		filename, len(parsed.Functions), len(parsed.Methods), len(parsed.Types), len(parsed.Interfaces))

	return parsed, nil
}

// parseImports extracts import statements
func (gp *GoParser) parseImports(file *ast.File, parsed *ParsedCode) {
	for _, importSpec := range file.Imports {
		imp := Import{
			Path: strings.Trim(importSpec.Path.Value, `"`),
			Line: gp.fileSet.Position(importSpec.Pos()).Line,
		}

		if importSpec.Name != nil {
			imp.Name = importSpec.Name.Name
		} else {
			// Extract package name from path
			parts := strings.Split(imp.Path, "/")
			imp.Name = parts[len(parts)-1]
		}

		// Check if it's a standard library import
		imp.Standard = gp.isStandardLibrary(imp.Path)

		parsed.Imports = append(parsed.Imports, imp)
	}
}

// parseGenDecl parses general declarations (types, constants, variables)
func (gp *GoParser) parseGenDecl(genDecl *ast.GenDecl, parsed *ParsedCode, docPkg *doc.Package) {
	switch genDecl.Tok {
	case token.TYPE:
		gp.parseTypeDecl(genDecl, parsed, docPkg)
	case token.CONST:
		gp.parseConstDecl(genDecl, parsed)
	case token.VAR:
		gp.parseVarDecl(genDecl, parsed)
	}
}

// parseTypeDecl parses type declarations
func (gp *GoParser) parseTypeDecl(genDecl *ast.GenDecl, parsed *ParsedCode, docPkg *doc.Package) {
	for _, spec := range genDecl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		typeDef := TypeDef{
			Name:      typeSpec.Name.Name,
			StartLine: gp.fileSet.Position(typeSpec.Pos()).Line,
			EndLine:   gp.fileSet.Position(typeSpec.End()).Line,
			DocString: gp.getDocString(genDecl.Doc),
		}

		switch t := typeSpec.Type.(type) {
		case *ast.StructType:
			typeDef.Kind = "struct"
			gp.parseStructFields(t, &typeDef)
		case *ast.InterfaceType:
			gp.parseInterface(typeSpec, t, parsed, docPkg)
			continue // Interface is handled separately
		case *ast.Ident:
			typeDef.Kind = "alias"
			typeDef.Underlying = t.Name
		case *ast.ArrayType, *ast.MapType, *ast.ChanType, *ast.FuncType:
			typeDef.Kind = "type"
			typeDef.Underlying = gp.typeToString(t)
		default:
			typeDef.Kind = "type"
			typeDef.Underlying = gp.typeToString(t)
		}

		parsed.Types = append(parsed.Types, typeDef)
	}
}

// parseStructFields parses struct fields
func (gp *GoParser) parseStructFields(structType *ast.StructType, typeDef *TypeDef) {
	if structType.Fields == nil {
		return
	}

	for _, field := range structType.Fields.List {
		fieldInfo := Field{
			Type:      gp.typeToString(field.Type),
			Line:      gp.fileSet.Position(field.Pos()).Line,
			DocString: gp.getDocString(field.Doc),
			Exported:  false,
		}

		if field.Tag != nil {
			fieldInfo.Tag = field.Tag.Value
		}

		// Handle embedded fields
		if len(field.Names) == 0 {
			fieldInfo.Embedded = true
			fieldInfo.Name = fieldInfo.Type
		} else {
			fieldInfo.Name = field.Names[0].Name
			fieldInfo.Exported = ast.IsExported(fieldInfo.Name)
		}

		typeDef.Fields = append(typeDef.Fields, fieldInfo)
	}
}

// parseInterface parses interface declarations
func (gp *GoParser) parseInterface(typeSpec *ast.TypeSpec, interfaceType *ast.InterfaceType, parsed *ParsedCode, docPkg *doc.Package) {
	interfaceDef := Interface{
		Name:      typeSpec.Name.Name,
		StartLine: gp.fileSet.Position(typeSpec.Pos()).Line,
		EndLine:   gp.fileSet.Position(typeSpec.End()).Line,
		DocString: gp.getDocString(nil), // Will be filled from docPkg
		Methods:   make([]InterfaceMethod, 0),
		Embedded:  make([]string, 0),
	}

	if interfaceType.Methods != nil {
		for _, method := range interfaceType.Methods.List {
			if len(method.Names) == 0 {
				// Embedded interface
				interfaceDef.Embedded = append(interfaceDef.Embedded, gp.typeToString(method.Type))
				continue
			}

			for _, name := range method.Names {
				if funcType, ok := method.Type.(*ast.FuncType); ok {
					interfaceMethod := InterfaceMethod{
						Name:       name.Name,
						Parameters: gp.parseParameters(funcType.Params),
						DocString:  gp.getDocString(method.Doc),
					}

					if funcType.Results != nil {
						interfaceMethod.Returns = gp.parseReturns(funcType.Results)
					}

					interfaceDef.Methods = append(interfaceDef.Methods, interfaceMethod)
				}
			}
		}
	}

	parsed.Interfaces = append(parsed.Interfaces, interfaceDef)
}

// parseConstDecl parses constant declarations
func (gp *GoParser) parseConstDecl(genDecl *ast.GenDecl, parsed *ParsedCode) {
	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		for i, name := range valueSpec.Names {
			constant := Constant{
				Name:      name.Name,
				Line:      gp.fileSet.Position(name.Pos()).Line,
				DocString: gp.getDocString(genDecl.Doc),
			}

			if valueSpec.Type != nil {
				constant.Type = gp.typeToString(valueSpec.Type)
			}

			if i < len(valueSpec.Values) {
				constant.Value = gp.exprToString(valueSpec.Values[i])
			}

			parsed.Constants = append(parsed.Constants, constant)
		}
	}
}

// parseVarDecl parses variable declarations
func (gp *GoParser) parseVarDecl(genDecl *ast.GenDecl, parsed *ParsedCode) {
	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		for i, name := range valueSpec.Names {
			variable := Variable{
				Name:      name.Name,
				Line:      gp.fileSet.Position(name.Pos()).Line,
				DocString: gp.getDocString(genDecl.Doc),
				IsGlobal:  true, // Top-level variables are global
			}

			if valueSpec.Type != nil {
				variable.Type = gp.typeToString(valueSpec.Type)
			}

			if i < len(valueSpec.Values) {
				variable.Value = gp.exprToString(valueSpec.Values[i])
			}

			parsed.Variables = append(parsed.Variables, variable)
		}
	}
}

// parseFuncDecl parses function declarations
func (gp *GoParser) parseFuncDecl(funcDecl *ast.FuncDecl, parsed *ParsedCode, docPkg *doc.Package) {
	if funcDecl.Recv != nil {
		// It's a method
		method := gp.parseMethod(funcDecl)
		parsed.Methods = append(parsed.Methods, method)
	} else {
		// It's a function
		function := gp.parseFunction(funcDecl)
		parsed.Functions = append(parsed.Functions, function)
	}
}

// parseFunction parses a function declaration
func (gp *GoParser) parseFunction(funcDecl *ast.FuncDecl) Function {
	function := Function{
		Name:        funcDecl.Name.Name,
		StartLine:   gp.fileSet.Position(funcDecl.Pos()).Line,
		EndLine:     gp.fileSet.Position(funcDecl.End()).Line,
		Visibility:  gp.getVisibility(funcDecl.Name.Name),
		DocString:   gp.getDocString(funcDecl.Doc),
		IsTest:      strings.HasPrefix(funcDecl.Name.Name, "Test"),
		IsBenchmark: strings.HasPrefix(funcDecl.Name.Name, "Benchmark"),
		CallsTo:     make([]string, 0),
	}

	// Parse function signature
	if funcDecl.Type != nil {
		function.Signature = gp.funcTypeToString(funcDecl.Type)
		function.Parameters = gp.parseParameters(funcDecl.Type.Params)

		if funcDecl.Type.Results != nil {
			function.Returns = gp.parseReturns(funcDecl.Type.Results)
		}
	}

	// Extract function body
	if funcDecl.Body != nil {
		function.Body = gp.blockToString(funcDecl.Body)
		function.Complexity = gp.calculateCyclomaticComplexity(funcDecl.Body)
		function.CallsTo = gp.extractFunctionCalls(funcDecl.Body)
	}

	return function
}

// parseMethod parses a method declaration
func (gp *GoParser) parseMethod(funcDecl *ast.FuncDecl) Method {
	method := Method{
		Function: gp.parseFunction(funcDecl),
	}

	// Parse receiver
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		recv := funcDecl.Recv.List[0]
		method.Receiver = Receiver{
			Type: gp.typeToString(recv.Type),
		}

		if len(recv.Names) > 0 {
			method.Receiver.Name = recv.Names[0].Name
		}

		// Check if receiver is a pointer
		if starExpr, ok := recv.Type.(*ast.StarExpr); ok {
			method.Receiver.IsPointer = true
			method.Receiver.Type = gp.typeToString(starExpr.X)
		}

		method.ReceiverType = method.Receiver.Type
	}

	return method
}

// parseParameters parses function parameters
func (gp *GoParser) parseParameters(fieldList *ast.FieldList) []Parameter {
	if fieldList == nil {
		return []Parameter{}
	}

	var parameters []Parameter
	for _, field := range fieldList.List {
		typeStr := gp.typeToString(field.Type)

		if len(field.Names) == 0 {
			// Anonymous parameter
			parameters = append(parameters, Parameter{
				Name: "",
				Type: typeStr,
			})
		} else {
			for _, name := range field.Names {
				parameters = append(parameters, Parameter{
					Name: name.Name,
					Type: typeStr,
				})
			}
		}
	}

	return parameters
}

// parseReturns parses function return values
func (gp *GoParser) parseReturns(fieldList *ast.FieldList) []Return {
	if fieldList == nil {
		return []Return{}
	}

	var returns []Return
	for _, field := range fieldList.List {
		typeStr := gp.typeToString(field.Type)

		if len(field.Names) == 0 {
			// Anonymous return
			returns = append(returns, Return{
				Type: typeStr,
			})
		} else {
			for _, name := range field.Names {
				returns = append(returns, Return{
					Name: name.Name,
					Type: typeStr,
				})
			}
		}
	}

	return returns
}

// parseComments extracts comments from the file
func (gp *GoParser) parseComments(file *ast.File, parsed *ParsedCode) {
	for _, commentGroup := range file.Comments {
		comment := Comment{
			Text:      commentGroup.Text(),
			StartLine: gp.fileSet.Position(commentGroup.Pos()).Line,
			EndLine:   gp.fileSet.Position(commentGroup.End()).Line,
		}

		// Determine comment type
		if len(commentGroup.List) == 1 && strings.HasPrefix(commentGroup.List[0].Text, "//") {
			comment.Type = "line"
		} else if strings.HasPrefix(commentGroup.List[0].Text, "/*") {
			comment.Type = "block"
		}

		// Check if it's a doc comment
		if strings.HasPrefix(strings.TrimSpace(comment.Text), "Package ") ||
			strings.Contains(comment.Text, "godoc") {
			comment.Type = "doc"
		}

		parsed.Comments = append(parsed.Comments, comment)
	}
}

// Helper methods

// typeToString converts an AST type to string
func (gp *GoParser) typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + gp.typeToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + gp.typeToString(t.Elt)
		}
		return "[" + gp.exprToString(t.Len) + "]" + gp.typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + gp.typeToString(t.Key) + "]" + gp.typeToString(t.Value)
	case *ast.ChanType:
		dir := ""
		switch t.Dir {
		case ast.SEND:
			dir = "chan<- "
		case ast.RECV:
			dir = "<-chan "
		default:
			dir = "chan "
		}
		return dir + gp.typeToString(t.Value)
	case *ast.FuncType:
		return gp.funcTypeToString(t)
	case *ast.SelectorExpr:
		return gp.typeToString(t.X) + "." + t.Sel.Name
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	default:
		return "unknown"
	}
}

// funcTypeToString converts a function type to string
func (gp *GoParser) funcTypeToString(funcType *ast.FuncType) string {
	var parts []string

	parts = append(parts, "func")

	if funcType.Params != nil {
		params := make([]string, 0)
		for _, param := range funcType.Params.List {
			typeStr := gp.typeToString(param.Type)
			if len(param.Names) == 0 {
				params = append(params, typeStr)
			} else {
				for _, name := range param.Names {
					params = append(params, name.Name+" "+typeStr)
				}
			}
		}
		parts = append(parts, "("+strings.Join(params, ", ")+")")
	} else {
		parts = append(parts, "()")
	}

	if funcType.Results != nil {
		results := make([]string, 0)
		for _, result := range funcType.Results.List {
			typeStr := gp.typeToString(result.Type)
			if len(result.Names) == 0 {
				results = append(results, typeStr)
			} else {
				for _, name := range result.Names {
					results = append(results, name.Name+" "+typeStr)
				}
			}
		}
		if len(results) == 1 && !strings.Contains(results[0], " ") {
			parts = append(parts, " "+results[0])
		} else {
			parts = append(parts, " ("+strings.Join(results, ", ")+")")
		}
	}

	return strings.Join(parts, "")
}

// exprToString converts an expression to string
func (gp *GoParser) exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return e.Value
	case *ast.Ident:
		return e.Name
	case *ast.BinaryExpr:
		return gp.exprToString(e.X) + " " + e.Op.String() + " " + gp.exprToString(e.Y)
	case *ast.UnaryExpr:
		return e.Op.String() + gp.exprToString(e.X)
	case *ast.CallExpr:
		args := make([]string, len(e.Args))
		for i, arg := range e.Args {
			args[i] = gp.exprToString(arg)
		}
		return gp.exprToString(e.Fun) + "(" + strings.Join(args, ", ") + ")"
	case *ast.SelectorExpr:
		return gp.exprToString(e.X) + "." + e.Sel.Name
	default:
		return "complex_expr"
	}
}

// blockToString converts a block statement to string (simplified)
func (gp *GoParser) blockToString(block *ast.BlockStmt) string {
	if block == nil {
		return ""
	}

	lines := make([]string, 0)
	for _, stmt := range block.List {
		line := gp.fileSet.Position(stmt.Pos()).Line
		lines = append(lines, fmt.Sprintf("Line %d: %T", line, stmt))
	}

	return strings.Join(lines, "\n")
}

// getDocString extracts documentation string
func (gp *GoParser) getDocString(commentGroup *ast.CommentGroup) string {
	if commentGroup == nil {
		return ""
	}
	return strings.TrimSpace(commentGroup.Text())
}

// getVisibility determines if a name is exported (public) or not
func (gp *GoParser) getVisibility(name string) string {
	if ast.IsExported(name) {
		return "public"
	}
	return "private"
}

// isStandardLibrary checks if import path is from standard library
func (gp *GoParser) isStandardLibrary(path string) bool {
	standardPrefixes := []string{
		"archive/", "bufio", "builtin", "bytes", "compress/", "container/",
		"context", "crypto/", "database/", "debug/", "embed", "encoding/",
		"errors", "expvar", "flag", "fmt", "go/", "hash/", "html/", "image/",
		"index/", "io/", "log/", "math/", "mime/", "net/", "os", "path/",
		"plugin", "reflect", "regexp", "runtime/", "sort", "strconv",
		"strings", "sync/", "syscall", "testing/", "text/", "time",
		"unicode/", "unsafe",
	}

	for _, prefix := range standardPrefixes {
		if path == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// calculateCyclomaticComplexity calculates cyclomatic complexity of a function
func (gp *GoParser) calculateCyclomaticComplexity(block *ast.BlockStmt) int {
	if block == nil {
		return 1
	}

	complexity := 1 // Base complexity

	ast.Inspect(block, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt,
			*ast.TypeSwitchStmt, *ast.SelectStmt:
			complexity++
		case *ast.CaseClause:
			complexity++
		}
		return true
	})

	return complexity
}

// extractFunctionCalls extracts function calls from a block
func (gp *GoParser) extractFunctionCalls(block *ast.BlockStmt) []string {
	if block == nil {
		return []string{}
	}

	calls := make(map[string]bool)

	ast.Inspect(block, func(n ast.Node) bool {
		if callExpr, ok := n.(*ast.CallExpr); ok {
			if funcName := gp.getFunctionName(callExpr.Fun); funcName != "" {
				calls[funcName] = true
			}
		}
		return true
	})

	result := make([]string, 0, len(calls))
	for call := range calls {
		result = append(result, call)
	}
	sort.Strings(result)

	return result
}

// getFunctionName extracts function name from call expression
func (gp *GoParser) getFunctionName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return gp.exprToString(e.X) + "." + e.Sel.Name
	default:
		return ""
	}
}

// calculateComplexity calculates various complexity metrics
func (gp *GoParser) calculateComplexity(parsed *ParsedCode, content string) {
	lines := strings.Split(content, "\n")
	nonEmptyLines := 0

	for _, line := range lines {
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "//") {
			nonEmptyLines++
		}
	}

	totalComplexity := 0
	for _, function := range parsed.Functions {
		totalComplexity += function.Complexity
	}
	for _, method := range parsed.Methods {
		totalComplexity += method.Complexity
	}

	parsed.Complexity = CodeComplexity{
		CyclomaticComplexity: totalComplexity,
		LinesOfCode:          nonEmptyLines,
		FunctionCount:        len(parsed.Functions),
		TypeCount:            len(parsed.Types),
		InterfaceCount:       len(parsed.Interfaces),
	}
}

// extractDependencies extracts dependency information
func (gp *GoParser) extractDependencies(parsed *ParsedCode) {
	deps := make(map[string]bool)

	for _, imp := range parsed.Imports {
		if !imp.Standard {
			deps[imp.Path] = true
		}
	}

	parsed.Dependencies = make([]string, 0, len(deps))
	for dep := range deps {
		parsed.Dependencies = append(parsed.Dependencies, dep)
	}
	sort.Strings(parsed.Dependencies)
}

// addMetadata adds file metadata
func (gp *GoParser) addMetadata(parsed *ParsedCode, filename string) {
	parsed.Metadata["filename"] = filepath.Base(filename)
	parsed.Metadata["extension"] = filepath.Ext(filename)
	parsed.Metadata["language"] = "go"
	parsed.Metadata["parser_version"] = "1.0"
}
