// Package eval implements reproducible comparisons of agent research workflows.
package eval

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Broker gives agents access only to the resources allocated to their arm.
// It is a benchmark adapter, not a general-purpose Legidor MCP server.
type Broker struct {
	Root    string `json:"root"`
	Legidor string `json:"legidor,omitempty"`
	Cache   string `json:"cache,omitempty"`
	Fresh   bool   `json:"fresh,omitempty"`
	Log     string `json:"log,omitempty"`
}

// Read rejects absolute paths, traversal and symlink escapes.
func (b Broker) Read(name string) (string, error) {
	if filepath.IsAbs(name) || name == "" {
		return "", errors.New("expected a relative resource path")
	}
	root, err := filepath.EvalSymlinks(b.Root)
	if err != nil {
		return "", err
	}
	p, err := filepath.EvalSymlinks(filepath.Join(root, name))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("path outside allocated resources")
	}
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 256*1024+1))
	if len(data) > 256*1024 {
		return "", errors.New("resource exceeds 256 KiB")
	}
	return string(data), err
}

// Calculate evaluates bounded arithmetic, with no variables or function calls.
func Calculate(expression string) (string, error) {
	if len(expression) > 2048 {
		return "", errors.New("expression too long")
	}
	tree, err := parser.ParseExpr(expression)
	if err != nil {
		return "", err
	}
	var eval func(ast.Expr, int) (float64, error)
	eval = func(e ast.Expr, depth int) (float64, error) {
		if depth > 64 {
			return 0, errors.New("expression too deep")
		}
		switch n := e.(type) {
		case *ast.BasicLit:
			if n.Kind != token.INT && n.Kind != token.FLOAT {
				break
			}
			return strconv.ParseFloat(n.Value, 64)
		case *ast.ParenExpr:
			return eval(n.X, depth+1)
		case *ast.UnaryExpr:
			v, err := eval(n.X, depth+1)
			if err != nil {
				return 0, err
			}
			if n.Op == token.SUB {
				return -v, nil
			}
			if n.Op == token.ADD {
				return v, nil
			}
		case *ast.BinaryExpr:
			a, err := eval(n.X, depth+1)
			if err != nil {
				return 0, err
			}
			b, err := eval(n.Y, depth+1)
			if err != nil {
				return 0, err
			}
			switch n.Op {
			case token.ADD:
				return a + b, nil
			case token.SUB:
				return a - b, nil
			case token.MUL:
				return a * b, nil
			case token.QUO:
				if b != 0 {
					return a / b, nil
				}
			}
		}
		return 0, errors.New("only finite numbers, parentheses and + - * / are supported")
	}
	v, err := eval(tree, 0)
	if err != nil {
		return "", err
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "", errors.New("non-finite result")
	}
	return strconv.FormatFloat(v, 'f', -1, 64), nil
}

// ValidateLegidor prevents file access and administrative commands through the adapter.
func ValidateLegidor(args []string) error {
	if len(args) == 0 || len(args) > 40 {
		return errors.New("expected legidor arguments")
	}
	switch args[0] {
	case "get", "resolve", "search", "versions", "diff":
	case "check", "refs":
		if len(args) < 2 || args[1] != "-" {
			return errors.New("check and refs accept stdin only")
		}
	default:
		return errors.New("only read-only research commands are allowed")
	}
	values := map[string]bool{"--date": true, "--part": true, "--grep": true, "--cursor": true, "--in": true, "--limit": true, "--since": true, "--from": true, "--to": true, "--max-age": true}
	flags := map[string]bool{"--json": true, "--full": true, "--parts": true, "--links": true, "--parse-only": true, "--fresh": true, "--offline": true, "--help": true, "--diff": true}
	for i := 1; i < len(args); i++ {
		s := args[i]
		if s == "-" {
			continue
		}
		if strings.HasPrefix(s, "-") {
			if values[s] {
				i++
				if i >= len(args) {
					return fmt.Errorf("missing value for %s", s)
				}
			} else if !flags[s] {
				return fmt.Errorf("unsupported flag %s", s)
			}
		}
	}
	return nil
}

func (b Broker) call(ctx context.Context, name string, a json.RawMessage) (string, error) {
	switch name {
	case "read_resource":
		var p struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(a, &p); err != nil {
			return "", err
		}
		return b.Read(p.Path)
	case "calculate":
		var p struct {
			Expression string `json:"expression"`
		}
		if err := json.Unmarshal(a, &p); err != nil {
			return "", err
		}
		return Calculate(p.Expression)
	case "legidor":
		if b.Legidor == "" {
			return "", errors.New("legidor is not allocated to this arm")
		}
		var p struct {
			Args  []string `json:"args"`
			Stdin string   `json:"stdin"`
		}
		if err := json.Unmarshal(a, &p); err != nil {
			return "", err
		}
		if err := ValidateLegidor(p.Args); err != nil {
			return "", err
		}
		if b.Fresh {
			p.Args = append(p.Args, "--fresh")
		}
		ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, b.Legidor, p.Args...)
		cmd.Stdin = strings.NewReader(p.Stdin)
		cmd.Env = append(os.Environ(), "XDG_CACHE_HOME="+b.Cache, "LEGIDOR_CACHE_DIR="+b.Cache)
		out, err := cmd.CombinedOutput()
		result := string(out)
		if err != nil {
			result += "\nProcess status: " + err.Error()
		}
		return result, nil
	default:
		return "", errors.New("unknown tool")
	}
}

func tool(name, description string, properties map[string]any, required []string) map[string]any {
	return map[string]any{"name": name, "description": description, "inputSchema": map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}}
}
func (b Broker) tools() []any {
	str := map[string]any{"type": "string"}
	ts := []any{tool("calculate", "Evaluate arithmetic (+ - * / and parentheses) without running code.", map[string]any{"expression": str}, []string{"expression"}), tool("read_resource", "Read a file from the allocated resources. Use paths listed in the task, relative to the resource root. No other files are accessible.", map[string]any{"path": str}, []string{"path"})}
	if b.Legidor != "" {
		ts = append(ts, tool("legidor", "Run the legidor CLI with an argv array, without the executable name. Supports get, resolve, search, versions, diff, and refs/check with '-' and stdin. Other commands and arbitrary file access are unavailable.", map[string]any{"args": map[string]any{"type": "array", "items": str}, "stdin": str}, []string{"args"}))
	}
	return ts
}

// Serve implements the newline-delimited stdio MCP subset used by both CLIs.
func (b Broker) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	scan := bufio.NewScanner(in)
	scan.Buffer(make([]byte, 4096), 2*1024*1024)
	enc := json.NewEncoder(out)
	for scan.Scan() {
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(scan.Bytes(), &req); err != nil {
			return err
		}
		if len(req.ID) == 0 {
			continue
		}
		var result any
		var rpcErr any
		switch req.Method {
		case "initialize":
			result = map[string]any{"protocolVersion": "2024-11-05", "capabilities": map[string]any{"tools": map[string]any{}}, "serverInfo": map[string]any{"name": "legidor-eval", "version": "1"}}
		case "ping":
			result = map[string]any{}
		case "tools/list":
			result = map[string]any{"tools": b.tools()}
		case "resources/list":
			result = map[string]any{"resources": []any{}}
		case "resources/templates/list":
			result = map[string]any{"resourceTemplates": []any{}}
		case "tools/call":
			var p struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			}
			err := json.Unmarshal(req.Params, &p)
			var s string
			start := time.Now()
			if err == nil {
				s, err = b.call(ctx, p.Name, p.Arguments)
			}
			if err != nil {
				s = err.Error()
			}
			result = map[string]any{"content": []any{map[string]any{"type": "text", "text": s}}, "isError": err != nil}
			if b.Log != "" {
				f, e := os.OpenFile(b.Log, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
				if e != nil {
					return e
				}
				e = json.NewEncoder(f).Encode(map[string]any{"tool": p.Name, "arguments": p.Arguments, "duration_ms": time.Since(start).Milliseconds(), "error": err != nil, "result": s})
				f.Close()
				if e != nil {
					return e
				}
			}
		default:
			rpcErr = map[string]any{"code": -32601, "message": "method not found"}
		}
		response := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		if rpcErr != nil {
			response["error"] = rpcErr
		} else {
			response["result"] = result
		}
		if err := enc.Encode(response); err != nil {
			return err
		}
	}
	return scan.Err()
}
