package test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/waffo-com/waffo-go/v2/config"
)

const localGoModule = "github.com/waffo-com/waffo-go/v2"

var allowedReadmeFrameworkImports = map[string]bool{
	"github.com/gin-gonic/gin":    true,
	"github.com/gofiber/fiber/v2": true,
	"github.com/labstack/echo/v4": true,
}

var readmeFrameworkStubs = map[string]string{
	"github.com/gin-gonic/gin": `package gin

import "net/http"

type Engine struct{}
type Context struct{ Request *http.Request }

func (*Engine) POST(string, func(*Context)) {}
func (c *Context) GetHeader(key string) string { return c.Request.Header.Get(key) }
func (*Context) Header(string, string) {}
func (*Context) String(int, string, ...interface{}) {}
`,
	"github.com/gofiber/fiber/v2": `package fiber

const StatusBadRequest = 400

type App struct{}
type Ctx struct{}

func (*App) Post(string, func(*Ctx) error) {}
func (*Ctx) Body() []byte { return nil }
func (*Ctx) Get(string) string { return "" }
func (*Ctx) Set(string, string) {}
func (*Ctx) SendString(string) error { return nil }
func (c *Ctx) Status(int) *Ctx { return c }
`,
	"github.com/labstack/echo/v4": `package echo

import "net/http"

type Echo struct{}
type HandlerFunc func(Context) error
type Context interface {
	Request() *http.Request
	Response() *Response
	String(int, string) error
}
type Response struct{ header http.Header }

func (*Echo) POST(string, HandlerFunc) {}
func (r *Response) Header() http.Header {
	if r.header == nil { r.header = make(http.Header) }
	return r.header
}
`,
}

type readmeGoBlock struct {
	path    string
	heading string
	index   int
	source  string
}

func TestReadmeDemoOffline_GoCodeBlocksParseAndUseCurrentModule(t *testing.T) {
	expectedModule := expectedPublicGoModule(t)
	readmes := goReadmePaths(t)
	blockCounts := make([]int, 0, len(readmes))

	for _, readmePath := range readmes {
		content, err := os.ReadFile(readmePath)
		if err != nil {
			t.Fatalf("read %s: %v", readmePath, err)
		}
		for lineNumber, line := range strings.Split(string(content), "\n") {
			if strings.Contains(line, "github.com/waffo-com/waffo-go") && !strings.Contains(line, expectedModule) {
				t.Errorf("%s:%d uses a public module path other than %s: %s", readmePath, lineNumber+1, expectedModule, line)
			}
			if strings.HasPrefix(strings.TrimSpace(line), "go get github.com/waffo-com/waffo-go") && strings.TrimSpace(line) != "go get "+expectedModule {
				t.Errorf("%s:%d installs a public module other than %s: %s", readmePath, lineNumber+1, expectedModule, line)
			}
		}

		blocks := extractReadmeGoBlocks(t, readmePath, content)
		blockCounts = append(blockCounts, len(blocks))
		if len(blocks) < 25 {
			t.Errorf("%s only has %d Go code blocks; expected at least 25", readmePath, len(blocks))
		}

		for _, block := range blocks {
			source := syntaxSource(block.source)
			file, err := parser.ParseFile(token.NewFileSet(), block.path, source, parser.AllErrors)
			if err != nil {
				t.Errorf("%s Go block %d (%s) is not valid Go syntax: %v", block.path, block.index, block.heading, err)
				continue
			}
			validateReadmeImports(t, block, file, expectedModule)
		}
	}

	for index := 0; index < len(blockCounts); index += 2 {
		if blockCounts[index] != blockCounts[index+1] {
			t.Errorf("%s has %d Go blocks while %s has %d", readmes[index], blockCounts[index], readmes[index+1], blockCounts[index+1])
		}
	}
}

func TestReadmeDemoOffline_GoCodeBlocksCompile(t *testing.T) {
	expectedModule := expectedPublicGoModule(t)
	tempDir := t.TempDir()
	writeReadmeCompileModule(t, tempDir)

	compiled := 0
	totalBlocks := 0
	for readmeIndex, readmePath := range goReadmePaths(t) {
		content, err := os.ReadFile(readmePath)
		if err != nil {
			t.Fatalf("read %s: %v", readmePath, err)
		}
		locale := "en"
		if strings.Contains(readmePath, "_CN") {
			locale = "zh"
		}

		for _, block := range extractReadmeGoBlocks(t, readmePath, content) {
			totalBlocks++
			parsed, err := parser.ParseFile(token.NewFileSet(), block.path, syntaxSource(block.source), parser.AllErrors)
			if err != nil {
				t.Fatalf("parse %s block %d before compile: %v", readmePath, block.index, err)
			}
			validateReadmeImports(t, block, parsed, expectedModule)

			packageDir := filepath.Join(tempDir, fmt.Sprintf("readme_%d_%s_%02d", readmeIndex, locale, block.index))
			if err := os.Mkdir(packageDir, 0o755); err != nil {
				t.Fatalf("create compile package for %s block %d: %v", readmePath, block.index, err)
			}
			source := compilableReadmeSource(block.source, expectedModule)
			if err := os.WriteFile(filepath.Join(packageDir, "demo.go"), []byte(source), 0o644); err != nil {
				t.Fatalf("write compile package for %s block %d: %v", readmePath, block.index, err)
			}
			compiled++
		}
	}

	if compiled != totalBlocks {
		t.Fatalf("selected %d of %d README Go blocks for compilation", compiled, totalBlocks)
	}
	output, err := runReadmeCompiler(tempDir)
	if err != nil {
		t.Fatalf("README Go blocks failed compiler validation:\n%s", output)
	}
}

func TestReadmeDemoOffline_BilingualGoAPISurfaceMatches(t *testing.T) {
	paths := goReadmePaths(t)
	for index := 0; index < len(paths); index += 2 {
		englishContent, err := os.ReadFile(paths[index])
		if err != nil {
			t.Fatalf("read %s: %v", paths[index], err)
		}
		chineseContent, err := os.ReadFile(paths[index+1])
		if err != nil {
			t.Fatalf("read %s: %v", paths[index+1], err)
		}

		english := readmeGoAPIFingerprint(t, extractReadmeGoBlocks(t, paths[index], englishContent))
		chinese := readmeGoAPIFingerprint(t, extractReadmeGoBlocks(t, paths[index+1], chineseContent))
		if !reflect.DeepEqual(english, chinese) {
			t.Errorf("%s and %s use different Go fields or methods:\n%s", paths[index], paths[index+1], fingerprintDifference(english, chinese))
		}
	}
}

func TestReadmeDemoOffline_CompilerRejectsUnknownSDKField(t *testing.T) {
	tempDir := t.TempDir()
	writeReadmeCompileModule(t, tempDir)
	badSource := compilableReadmeSource(`
resp, err := client.Order().Create(ctx, &order.CreateOrderParams{
    ResultCode: "not-a-request-field",
}, nil)
`, expectedPublicGoModule(t))
	packageDir := filepath.Join(tempDir, "invalid_field")
	if err := os.Mkdir(packageDir, 0o755); err != nil {
		t.Fatalf("create negative compile package: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packageDir, "demo.go"), []byte(badSource), 0o644); err != nil {
		t.Fatalf("write negative compile package: %v", err)
	}

	output, err := runReadmeCompiler(tempDir)
	if err == nil {
		t.Fatal("README compiler accepted an unknown SDK field")
	}
	if !strings.Contains(output, "ResultCode") {
		t.Fatalf("compiler failed for an unexpected reason:\n%s", output)
	}
}

func expectedPublicGoModule(t *testing.T) string {
	t.Helper()
	major := strings.Split(config.SDKVersion, ".")[0]
	if major == "" {
		t.Fatalf("invalid Go SDK version %q", config.SDKVersion)
	}
	if major == "1" {
		return "github.com/waffo-com/waffo-go"
	}
	return "github.com/waffo-com/waffo-go/v" + major
}

func goReadmePaths(t *testing.T) []string {
	t.Helper()
	paths := []string{"../README.md", "../README_CN.md"}
	if _, err := os.Stat("../../../README.md"); err == nil {
		paths = append(paths, "../../../README.md", "../../../README_CN.md")
	} else if !os.IsNotExist(err) {
		t.Fatalf("inspect monorepo README: %v", err)
	}
	return paths
}

func extractReadmeGoBlocks(t *testing.T, path string, content []byte) []readmeGoBlock {
	t.Helper()
	var blocks []readmeGoBlock
	var blockLines []string
	heading := ""
	inGoBlock := false

	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if !inGoBlock && strings.HasPrefix(trimmed, "#") {
			heading = trimmed
		}
		if !inGoBlock && trimmed == "```go" {
			inGoBlock = true
			blockLines = nil
			continue
		}
		if inGoBlock && trimmed == "```" {
			blocks = append(blocks, readmeGoBlock{
				path:    path,
				heading: heading,
				index:   len(blocks) + 1,
				source:  strings.Join(blockLines, "\n"),
			})
			inGoBlock = false
			continue
		}
		if inGoBlock {
			blockLines = append(blockLines, line)
		}
	}
	if inGoBlock {
		t.Fatalf("%s has an unclosed Go code block under %s", path, heading)
	}
	return blocks
}

func readmeGoAPIFingerprint(t *testing.T, blocks []readmeGoBlock) map[string]int {
	t.Helper()
	fingerprint := make(map[string]int)
	for _, block := range blocks {
		file, err := parser.ParseFile(token.NewFileSet(), block.path, syntaxSource(block.source), parser.AllErrors)
		if err != nil {
			t.Fatalf("parse %s block %d for bilingual comparison: %v", block.path, block.index, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch value := node.(type) {
			case *ast.SelectorExpr:
				fingerprint["selector:"+value.Sel.Name]++
			case *ast.KeyValueExpr:
				if field, ok := value.Key.(*ast.Ident); ok {
					fingerprint["field:"+field.Name]++
				}
			}
			return true
		})
	}
	return fingerprint
}

func fingerprintDifference(english, chinese map[string]int) string {
	keys := make(map[string]bool)
	for key := range english {
		keys[key] = true
	}
	for key := range chinese {
		keys[key] = true
	}
	ordered := make([]string, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Strings(ordered)

	var differences []string
	for _, key := range ordered {
		if english[key] != chinese[key] {
			differences = append(differences, fmt.Sprintf("%s: en=%d, zh=%d", key, english[key], chinese[key]))
		}
	}
	return strings.Join(differences, "\n")
}

func syntaxSource(source string) string {
	if strings.HasPrefix(strings.TrimSpace(source), "package ") {
		return source
	}
	imports, body := splitLeadingImports(source)
	if hasTopLevelDeclaration(body) {
		return "package readmedemo\n\n" + imports + "\n" + body
	}
	return "package readmedemo\n\n" + imports + "\nfunc demo() {\n" + body + "\n}\n"
}

func splitLeadingImports(source string) (string, string) {
	lines := strings.Split(source, "\n")
	first := 0
	for first < len(lines) && strings.TrimSpace(lines[first]) == "" {
		first++
	}
	if first == len(lines) || !strings.HasPrefix(strings.TrimSpace(lines[first]), "import ") {
		return "", source
	}

	last := first
	if strings.TrimSpace(lines[first]) == "import (" {
		for last++; last < len(lines); last++ {
			if strings.TrimSpace(lines[last]) == ")" {
				break
			}
		}
		if last == len(lines) {
			return strings.Join(lines[first:], "\n"), strings.Join(lines[:first], "\n")
		}
	}
	imports := strings.Join(lines[first:last+1], "\n")
	body := strings.Join(append(lines[:first], lines[last+1:]...), "\n")
	return imports, body
}

func hasTopLevelDeclaration(source string) bool {
	for _, line := range strings.Split(source, "\n") {
		if line != strings.TrimLeft(line, " \t") {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "func ") || strings.HasPrefix(trimmed, "type ") || strings.HasPrefix(trimmed, "var ") || strings.HasPrefix(trimmed, "const ") {
			return true
		}
	}
	return false
}

func validateReadmeImports(t *testing.T, block readmeGoBlock, file *ast.File, expectedModule string) {
	t.Helper()
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Errorf("%s block %d (%s) has invalid import %s", block.path, block.index, block.heading, spec.Path.Value)
			continue
		}
		if strings.HasPrefix(importPath, "github.com/waffo-com/waffo-go") {
			if importPath != expectedModule && !strings.HasPrefix(importPath, expectedModule+"/") {
				t.Errorf("%s block %d (%s) imports %s instead of %s", block.path, block.index, block.heading, importPath, expectedModule)
				continue
			}
			localPath := strings.TrimPrefix(importPath, expectedModule)
			if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(strings.TrimPrefix(localPath, "/")))); err != nil {
				t.Errorf("%s block %d (%s) imports SDK package %s with no local source directory", block.path, block.index, block.heading, importPath)
			}
			continue
		}
		if isThirdPartyImport(importPath) {
			if !allowedReadmeFrameworkImports[importPath] {
				t.Errorf("%s block %d (%s) adds unvalidated third-party import %s", block.path, block.index, block.heading, importPath)
			}
		}
	}
}

func isThirdPartyImport(importPath string) bool {
	firstPart := strings.Split(importPath, "/")[0]
	return strings.Contains(firstPart, ".")
}

func compilableReadmeSource(source, publicModule string) string {
	source = strings.ReplaceAll(source, publicModule, localGoModule)
	if strings.HasPrefix(strings.TrimSpace(source), "package ") {
		return source
	}

	imports, body := splitLeadingImports(source)
	if hasTopLevelDeclaration(body) {
		return "package readmedemo\n\n" + imports + "\n" + body + "\n"
	}

	// API sections intentionally show statement fragments. Supply their shared
	// client/import context while preserving the README statements verbatim.
	return fmt.Sprintf(`package readmedemo

import (
    "context"
    "errors"
    "fmt"
    "log"
    "time"

    waffo "%[1]s"
    "%[1]s/config"
    "%[1]s/core"
    waffoErrors "%[1]s/errors"
    "%[1]s/net"
    "%[1]s/types"
    "%[1]s/types/merchant"
    "%[1]s/types/order"
    "%[1]s/types/refund"
    "%[1]s/types/subscription"
    "%[1]s/types/wallet"
)

var (
    _ = context.Background
    _ = errors.As
    _ = fmt.Printf
    _ = log.Printf
    _ = time.Now
    _ = config.Sandbox
    _ = core.PaymentNotification{}
    _ = waffoErrors.WaffoError{}
    _ = net.NewDefaultHttpTransport
    _ types.ExtraParams
    _ merchant.InquiryMerchantConfigParams
    _ refund.InquiryRefundParams
    _ subscription.CreateSubscriptionParams
    _ wallet.WalletInquiryParams
)

func demo(client *waffo.Waffo%[4]s, params *order.CreateOrderParams) {
%[2]s
%[3]s
}
`, localGoModule, body, unusedReadmeVariables(body), readmeContextParameter(body))
}

func readmeContextParameter(source string) string {
	if strings.Contains(source, "ctx :=") {
		return ""
	}
	return ", ctx context.Context"
}

func unusedReadmeVariables(source string) string {
	var lines []string
	for _, variable := range []string{"resp", "err", "client", "cfg", "handler", "transport"} {
		if strings.Contains(source, variable+", err :=") || strings.Contains(source, variable+" :=") {
			lines = append(lines, "    _ = "+variable)
		}
	}
	if strings.Contains(source, ", err :=") {
		lines = append(lines, "    _ = err")
	}
	return strings.Join(lines, "\n")
}

func writeReadmeCompileModule(t *testing.T, tempDir string) {
	t.Helper()
	packageRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve Go SDK package root: %v", err)
	}
	moduleVersion := "v0.0.0"
	if segment := filepath.Base(localGoModule); strings.HasPrefix(segment, "v") {
		if major, err := strconv.Atoi(segment[1:]); err == nil && major >= 2 {
			moduleVersion = fmt.Sprintf("v%d.0.0", major)
		}
	}
	goMod := fmt.Sprintf(`module readmedemo

go 1.20

require (
	%[1]s %[3]s
	github.com/gin-gonic/gin v1.0.0
	github.com/gofiber/fiber/v2 v2.0.0
	github.com/labstack/echo/v4 v4.0.0
)

replace %[1]s => %[2]s
replace github.com/gin-gonic/gin => ./framework/gin
replace github.com/gofiber/fiber/v2 => ./framework/fiber
replace github.com/labstack/echo/v4 => ./framework/echo
`, localGoModule, strconv.Quote(filepath.ToSlash(packageRoot)), moduleVersion)
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write README compile go.mod: %v", err)
	}
	writeReadmeFrameworkStubs(t, tempDir)
}

func writeReadmeFrameworkStubs(t *testing.T, tempDir string) {
	t.Helper()
	for modulePath, source := range readmeFrameworkStubs {
		name := filepath.Base(modulePath)
		if name == "v2" || name == "v4" {
			name = filepath.Base(filepath.Dir(modulePath))
		}
		packageDir := filepath.Join(tempDir, "framework", name)
		if err := os.MkdirAll(packageDir, 0o755); err != nil {
			t.Fatalf("create README framework stub %s: %v", modulePath, err)
		}
		goMod := "module " + modulePath + "\n\ngo 1.20\n"
		if err := os.WriteFile(filepath.Join(packageDir, "go.mod"), []byte(goMod), 0o644); err != nil {
			t.Fatalf("write README framework stub module %s: %v", modulePath, err)
		}
		if err := os.WriteFile(filepath.Join(packageDir, "stub.go"), []byte(source), 0o644); err != nil {
			t.Fatalf("write README framework stub source %s: %v", modulePath, err)
		}
	}
}

func runReadmeCompiler(tempDir string) (string, error) {
	command := exec.Command("go", "test", "./...")
	command.Dir = tempDir
	command.Env = append(os.Environ(), "GOPROXY=off", "GOWORK=off")
	output, err := command.CombinedOutput()
	return string(output), err
}
