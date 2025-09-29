package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/decls"
	"github.com/google/cel-go/common/types"
	celext "github.com/google/cel-go/ext"
	"github.com/spf13/cobra"
	triggersv1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggerscel "github.com/tektoncd/triggers/pkg/interceptors/cel"
)

var (
	rootCmd = &cobra.Command{
		Use:   "cel-eval",
		Short: "Tekton CEL interceptor evaluator",
		Run:   rootRun,
	}

	expressionPath string
	httpPath       string
)

func init() {
	rootCmd.Flags().StringVarP(&expressionPath, "expression", "e", "", "Expression to evaluate")
	rootCmd.Flags().StringVarP(&httpPath, "http-request", "r", "", "Path to HTTP request")
	if err := rootCmd.MarkFlagRequired("expression"); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if err := rootCmd.MarkFlagRequired("http-request"); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

// revive:disable:unused-parameter

func rootRun(cmd *cobra.Command, args []string) {
	if err := evalCEL(cmd.Context(), os.Stdout, expressionPath, httpPath); err != nil {
		log.Fatal(err)
	}
}

type secretGetter struct{}

func (sg secretGetter) Get(ctx context.Context, triggerNS string, sr *triggersv1beta1.SecretRef) ([]byte, error) {
	return nil, nil
}

func evalCEL(ctx context.Context, w io.Writer, expressionPath, httpPath string) error {
	// Read expression
	expression, err := readExpression(expressionPath)
	if err != nil {
		return fmt.Errorf("error reading HTTP file: %w", err)
	}

	// Read HTTP request.
	r, body, err := readHTTP(httpPath)
	if err != nil {
		return fmt.Errorf("error reading HTTP file: %w", err)
	}

	evalContext, err := makeEvalContext(body, r.Header, r.URL.String(), map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("error making eval context: %w", err)
	}

	mapStrDyn := types.NewMapType(types.StringType, types.DynType)
	env, err := cel.NewEnv(
		triggerscel.Triggers(ctx, "default", secretGetter{}),
		celext.Strings(),
		celext.Encoders(),
		celext.Sets(),
		celext.Lists(),
		celext.Math(),
		cel.VariableDecls(
			decls.NewVariable("body", mapStrDyn),
			decls.NewVariable("header", mapStrDyn),
			decls.NewVariable("extensions", mapStrDyn),
			decls.NewVariable("requestURL", types.StringType),
		))
	if err != nil {
		log.Fatal(err)
	}

	parsed, issues := env.Parse(expression)
	if issues != nil && issues.Err() != nil {
		return fmt.Errorf("failed to parse expression %#v: %w", expression, issues.Err())
	}

	checked, issues := env.Check(parsed)
	if issues != nil && issues.Err() != nil {
		return fmt.Errorf("expression %#v check failed: %w", expression, issues.Err())
	}

	prg, err := env.Program(checked)
	if err != nil {
		return fmt.Errorf("expression %#v failed to create a Program: %w", expression, err)
	}

	out, _, err := prg.Eval(evalContext)
	if err != nil {
		return fmt.Errorf("expression %#v failed to evaluate: %w", expression, err)
	}

	fmt.Fprint(w, out)

	return nil
}

func readExpression(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("error opening file: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("error reading from file: %w", err)
	}

	return string(data), nil
}

func readHTTP(path string) (*http.Request, []byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening file: %w", err)
	}
	defer f.Close()

	// Read the entire file content first
	fileContent, err := io.ReadAll(f)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading file: %w", err)
	}

	// Split into headers and body parts
	parts := strings.SplitN(string(fileContent), "\n\n", 2)
	if len(parts) != 2 {
		parts = strings.SplitN(string(fileContent), "\r\n\r\n", 2)
	}

	var headerPart, bodyPart string
	if len(parts) == 2 {
		headerPart = parts[0]
		bodyPart = parts[1]
	} else {
		headerPart = string(fileContent)
		bodyPart = ""
	}

	// Auto compute and fill the content length field if it is not present
	if len(bodyPart) > 0 && !strings.Contains(headerPart, "Content-Length:") {
		headerPart += fmt.Sprintf("\nContent-Length: %d", len(bodyPart))
	}

	// Reconstruct the HTTP request with the Content-Length header
	reconstructedContent := headerPart + "\n\n" + bodyPart

	req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(reconstructedContent)))
	if err != nil {
		return nil, nil, fmt.Errorf("error reading request: %w", err)
	}
	defer req.Body.Close()
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading HTTP body: %w", err)
	}

	return req, body, nil
}

func makeEvalContext(body []byte, h http.Header, url string, extensions map[string]interface{}) (map[string]interface{}, error) {
	var jsonMap map[string]interface{}
	err := json.Unmarshal(body, &jsonMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the body as JSON: %w", err)
	}
	return map[string]interface{}{
		"body":       jsonMap,
		"header":     h,
		"requestURL": url,
		"extensions": extensions,
	}, nil
}

// Execute runs the command.
func Execute() error {
	return rootCmd.Execute()
}
