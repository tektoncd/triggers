/*
Copyright 2019 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	gh "github.com/google/go-github/v31/github"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/codes"
)

var _ triggersv1.InterceptorInterface = (*Interceptor)(nil)

var acceptedEventTypes = []string{"pull_request", "push"}

type testURLKey string

const (
	changedFilesExtensionsKey            = "changed_files"
	testURL                   testURLKey = "TESTURL"
)

// ErrInvalidContentType is returned when the content-type is not a JSON body.
var ErrInvalidContentType = errors.New("form parameter encoding not supported, please change the hook to send JSON payloads")

type Interceptor struct {
	SecretGetter interceptors.SecretGetter
}

type payloadDetails struct {
	PrNumber     int
	Owner        string
	Repository   string
	ChangedFiles string
}

func NewInterceptor(sg interceptors.SecretGetter) *Interceptor {
	return &Interceptor{
		SecretGetter: sg,
	}
}

func (w *Interceptor) Process(ctx context.Context, r *triggersv1.InterceptorRequest) *triggersv1.InterceptorResponse {
	headers := interceptors.Canonical(r.Header)
	if v := headers.Get("Content-Type"); v == "application/x-www-form-urlencoded" {
		return interceptors.Fail(codes.InvalidArgument, ErrInvalidContentType.Error())
	}

	p := triggersv1.GitHubInterceptor{}
	if err := interceptors.UnmarshalParams(r.InterceptorParams, &p); err != nil {
		return interceptors.Failf(codes.InvalidArgument, "failed to parse interceptor params: %v", err)
	}

	actualEvent := headers.Get("X-GitHub-Event")

	// Check if the event type is in the allow-list
	if p.EventTypes != nil {
		isAllowed := false
		for _, allowedEvent := range p.EventTypes {
			if actualEvent == allowedEvent {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			return interceptors.Failf(codes.FailedPrecondition, "event type %s is not allowed", actualEvent)
		}
	}

	// Next validate secrets
	if p.SecretRef != nil {
		// Check the secret to see if it is empty
		if p.SecretRef.SecretKey == "" {
			return interceptors.Fail(codes.FailedPrecondition, "github interceptor secretRef.secretKey is empty")
		}
		header := headers.Get("X-Hub-Signature-256")
		if header == "" {
			header = headers.Get("X-Hub-Signature")
		}
		if header == "" {
			return interceptors.Fail(codes.FailedPrecondition, "Must set X-Hub-Signature-256 or X-Hub-Signature header")
		}

		if r.Context == nil {
			return interceptors.Failf(codes.InvalidArgument, "no request context passed")
		}

		ns, _ := triggersv1.ParseTriggerID(r.Context.TriggerID)
		secretToken, err := w.SecretGetter.Get(ctx, ns, p.SecretRef)
		if err != nil {
			return interceptors.Failf(codes.FailedPrecondition, "error getting secret: %v", err)
		}

		if err := gh.ValidateSignature(header, []byte(r.Body), secretToken); err != nil {
			return interceptors.Fail(codes.FailedPrecondition, err.Error())
		}
	}

	if p.AddChangedFiles.Enabled {
		shouldAddChangedFiles := false
		for _, allowedEvent := range acceptedEventTypes {
			if actualEvent == allowedEvent {
				shouldAddChangedFiles = true
				break
			}
		}
		if !shouldAddChangedFiles {
			return &triggersv1.InterceptorResponse{
				Continue: true,
			}
		}

		if r.Context == nil {
			return interceptors.Failf(codes.InvalidArgument, "no request context passed")
		}

		secretToken, err := w.getGithubTokenSecret(ctx, r, p)
		if err != nil {
			return interceptors.Failf(codes.FailedPrecondition, "error getting secret: %v", err)
		}

		payload, err := parseBodyForChangedFiles(r.Body, actualEvent)
		if err != nil {
			return interceptors.Failf(codes.FailedPrecondition, "error parsing body: %v", err)
		}

		var changedFiles string
		if actualEvent == "pull_request" {
			changedFiles, err = getChangedFilesFromPr(ctx, payload, headers.Get("X-Github-Enterprise-Host"), secretToken)
			if err != nil {
				return interceptors.Failf(codes.FailedPrecondition, "error getting changed files: %v", err)
			}
		} else {
			changedFiles = payload.ChangedFiles
		}

		return &triggersv1.InterceptorResponse{
			Extensions: map[string]interface{}{
				changedFilesExtensionsKey: changedFiles,
			},
			Continue: true,
		}

	}

	return &triggersv1.InterceptorResponse{
		Continue: true,
	}
}

func (w *Interceptor) getGithubTokenSecret(ctx context.Context, r *triggersv1.InterceptorRequest, p triggersv1.GitHubInterceptor) (string, error) {
	if p.AddChangedFiles.PersonalAccessToken == nil {
		return "", nil
	}
	if p.AddChangedFiles.PersonalAccessToken.SecretKey == "" {
		return "", fmt.Errorf("github interceptor githubToken.secretKey is empty")
	}
	ns, _ := triggersv1.ParseTriggerID(r.Context.TriggerID)
	secretToken, err := w.SecretGetter.Get(ctx, ns, p.AddChangedFiles.PersonalAccessToken)
	if err != nil {
		return "", err
	}
	return string(secretToken), nil
}

func parseBodyForChangedFiles(body string, eventType string) (payloadDetails, error) {
	results := payloadDetails{}
	if body == "" {
		return results, fmt.Errorf("body is empty")
	}

	var jsonMap map[string]interface{}
	err := json.Unmarshal([]byte(body), &jsonMap)
	if err != nil {
		return results, err
	}

	var prNum int
	_, ok := jsonMap["number"]
	if ok {
		prNum = int(jsonMap["number"].(float64))
	} else {
		if eventType == "pull_request" {
			return results, fmt.Errorf("pull_request body missing 'number' field")
		}
		prNum = -1
	}

	repoSection, ok := jsonMap["repository"].(map[string]interface{})
	if !ok {
		return results, fmt.Errorf("payload body missing 'repository' field")
	}

	fullName, ok := repoSection["full_name"].(string)
	if !ok {
		return results, fmt.Errorf("payload body missing 'repository.full_name' field")
	}

	changedFiles := []string{}

	commitsSection, ok := jsonMap["commits"].([]interface{})
	if ok {

		for _, commit := range commitsSection {
			addedFiles, ok := commit.(map[string]interface{})["added"].([]interface{})
			if !ok {
				return results, fmt.Errorf("payload body missing 'commits.*.added' field")
			}

			modifiedFiles, ok := commit.(map[string]interface{})["modified"].([]interface{})
			if !ok {
				return results, fmt.Errorf("payload body missing 'commits.*.modified' field")
			}

			removedFiles, ok := commit.(map[string]interface{})["removed"].([]interface{})
			if !ok {
				return results, fmt.Errorf("payload body missing 'commits.*.removed' field")
			}
			for _, fileName := range addedFiles {
				changedFiles = append(changedFiles, fmt.Sprintf("%v", fileName))
			}

			for _, fileName := range modifiedFiles {
				changedFiles = append(changedFiles, fmt.Sprintf("%v", fileName))
			}

			for _, fileName := range removedFiles {
				changedFiles = append(changedFiles, fmt.Sprintf("%v", fileName))
			}
		}
	}

	results = payloadDetails{
		PrNumber:     prNum,
		Owner:        strings.Split(fullName, "/")[0],
		Repository:   strings.Split(fullName, "/")[1],
		ChangedFiles: strings.Join(changedFiles, ","),
	}
	return results, nil
}

func getChangedFilesFromPr(ctx context.Context, payload payloadDetails, enterpriseBaseURL string, token string) (string, error) {

	changedFiles := []string{}

	client, err := makeClient(ctx, enterpriseBaseURL, token)
	if err != nil {
		return "", err
	}

	opt := &gh.ListOptions{PerPage: 100}
	for {
		files, resp, err := client.PullRequests.ListFiles(ctx, payload.Owner, payload.Repository, payload.PrNumber, opt)
		if err != nil {
			return "", err
		}
		for _, file := range files {
			changedFiles = append(changedFiles, *file.Filename)
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return strings.Join(changedFiles, ","), nil
}

func makeClient(ctx context.Context, enterpriseBaseURL string, token string) (*gh.Client, error) {

	var httpClient *http.Client
	var client *gh.Client
	var err error

	if token != "" {
		tokenSource := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		httpClient = oauth2.NewClient(ctx, tokenSource)
	} else {
		httpClient = nil
	}

	testingURL := ""
	if ctx.Value(testURL) != nil {
		testingURL = fmt.Sprintf("%v", ctx.Value(testURL))
	}

	if enterpriseBaseURL != "" || testingURL != "" {
		enterpriseBaseURL = "https://" + enterpriseBaseURL
		if testingURL != "" {
			enterpriseBaseURL = testingURL
		}

		client, err = gh.NewEnterpriseClient(enterpriseBaseURL, enterpriseBaseURL, httpClient)
		if err != nil {
			return client, err
		}
	} else {
		client = gh.NewClient(httpClient)
	}
	return client, nil
}
