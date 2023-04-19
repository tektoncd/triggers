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

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
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
	if err := evalCEL(os.Stdout, expressionPath, httpPath); err != nil {
		log.Fatal(err)
	}
}

type secretGetter struct{}

func (sg secretGetter) Get(ctx context.Context, triggerNS string, sr *triggersv1beta1.SecretRef) ([]byte, error) {
	return nil, nil
}

func evalCEL(w io.Writer, expressionPath, httpPath string) error {
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

	mapStrDyn := decls.NewMapType(decls.String, decls.Dyn)
	env, err := cel.NewEnv(
		triggerscel.Triggers(context.Background(), "default", secretGetter{}),
		celext.Strings(),
		celext.Encoders(),
		cel.Declarations(
			decls.NewVar("body", mapStrDyn),
			decls.NewVar("header", mapStrDyn),
			decls.NewVar("extensions", mapStrDyn),
			decls.NewVar("requestURL", decls.String),
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

	req, err := http.ReadRequest(bufio.NewReader(f))
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
