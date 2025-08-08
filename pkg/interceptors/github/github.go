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
	"regexp"
	"strings"

	gh "github.com/google/go-github/v31/github"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/codes"
	"gopkg.in/yaml.v2"
)

var _ triggersv1.InterceptorInterface = (*InterceptorImpl)(nil)

const pullRequest = "pull_request"

var acceptedEventTypes = []string{pullRequest, "push"}

type testURLKey string

const (
	changedFilesExtensionsKey            = "changed_files"
	testURL                   testURLKey = "TESTURL"
	OKToTestCommentRegexp                = `(^|\n)\/ok-to-test(\r\n|\r|\n|$)`
)

// In a pull request, these are the only two events that should trigger a PipelineRun/TaskRun
var ownersEventTypes = []string{pullRequest, "issue_comment"}

// ErrInvalidContentType is returned when the content-type is not a JSON body.
var ErrInvalidContentType = errors.New("form parameter encoding not supported, please change the hook to send JSON payloads")

type InterceptorImpl struct {
	SecretGetter interceptors.SecretGetter
}

type payloadDetails struct {
	PrNumber     int
	Owner        string
	Repository   string
	ChangedFiles string
}

type OwnersPayloadDetails struct {
	PrNumber         int
	Sender           string
	Owner            string
	Repository       string
	IssueCommentBody string
}

type OwnersConfig struct {
	Approvers []string `json:"approvers,omitempty"`
	Reviewers []string `json:"reviewers,omitempty"`
}

func NewInterceptor(sg interceptors.SecretGetter) *InterceptorImpl {
	return &InterceptorImpl{
		SecretGetter: sg,
	}
}

// InterceptorParams provides a webhook to intercept and pre-process events
type InterceptorParams struct {
	SecretRef *triggersv1.SecretRef `json:"secretRef,omitempty"`
	// +listType=atomic
	EventTypes      []string        `json:"eventTypes,omitempty"`
	AddChangedFiles AddChangedFiles `json:"addChangedFiles,omitempty"`
	GithubOwners    Owners          `json:"githubOwners,omitempty"`
}

type CheckType string

const (
	// Set the checkType to orgMembers to allow org members to submit or comment on PR to proceed
	OrgMembers CheckType = "orgMembers"
	// Set the checkType to repoMembers to allow repo members to submit or comment on PR to proceed
	RepoMembers CheckType = "repoMembers"
	// Set the checkType to all if both repo members or org members can submit or comment on PR to proceed
	All CheckType = "all"
	// Set the checkType to none if neither of repo members or org members can not submit or comment on PR to proceed
	None CheckType = "none"
)

type Owners struct {
	Enabled bool `json:"enabled,omitempty"`
	// This param/variable is required for private repos or when checkType is set to orgMembers or repoMembers or all
	PersonalAccessToken *triggersv1.SecretRef `json:"personalAccessToken,omitempty"`
	// Set the value to one of the supported values (orgMembers, repoMembers, both, none)
	CheckType CheckType `json:"checkType,omitempty"`
}

type AddChangedFiles struct {
	Enabled             bool                  `json:"enabled,omitempty"`
	PersonalAccessToken *triggersv1.SecretRef `json:"personalAccessToken,omitempty"`
}

func (w *InterceptorImpl) Process(ctx context.Context, r *triggersv1.InterceptorRequest) *triggersv1.InterceptorResponse {
	headers := interceptors.Canonical(r.Header)
	if v := headers.Get("Content-Type"); v == "application/x-www-form-urlencoded" {
		return interceptors.Fail(codes.InvalidArgument, ErrInvalidContentType.Error())
	}

	p := InterceptorParams{}
	if err := interceptors.UnmarshalParams(r.InterceptorParams, &p); err != nil {
		return interceptors.Failf(codes.InvalidArgument, "failed to parse interceptor params: %v", err)
	}

	actualEvent := headers.Get("X-Github-Event")

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
		if actualEvent == pullRequest {
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

	// For event types pull_request, issue_comment check github owners approval is required
	// User can specify both event type or any one of them
	if p.GithubOwners.Enabled {
		ownerCheckAllowed := false
		for _, allowedEvent := range ownersEventTypes {
			if actualEvent == allowedEvent {
				ownerCheckAllowed = true
				break
			}
		}
		if !ownerCheckAllowed {
			return &triggersv1.InterceptorResponse{
				Continue: true,
			}
		}
		ghToken, err := w.getPersonalAccessTokenSecret(ctx, r, p)
		if err != nil {
			return interceptors.Failf(codes.FailedPrecondition, "error getting github token: %v", err)
		}
		if ghToken == "" && (p.GithubOwners.CheckType != "none") {
			return interceptors.Fail(codes.FailedPrecondition, "checkType is set to check org or repo members but no personalAccessToken was supplied")
		}
		// The X-Github-Enterprise-Host header only exists when the webhook comes from a github enterprise
		// server and is left empty for regular hosted Github
		enterpriseBaseURL := headers.Get("X-Github-Enterprise-Host")
		client, err := makeClient(ctx, enterpriseBaseURL, ghToken)
		if err != nil {
			return interceptors.Failf(codes.FailedPrecondition, "error making client: %v", err)
		}
		payload, err := parseBodyForOwners(r.Body, actualEvent)
		if err != nil {
			return interceptors.Failf(codes.FailedPrecondition, "error parsing body: %v", err)
		}
		allowed, err := checkOwnershipAndMembership(ctx, payload, p, client)
		if err != nil {
			return interceptors.Failf(codes.FailedPrecondition, "error checking owner verification: %v", err)
		}

		if allowed && actualEvent == pullRequest {
			return &triggersv1.InterceptorResponse{
				Continue: true,
			}
		}

		commentAllowed, err := okToTestFromAnOwner(ctx, payload, p, client)
		if err != nil {
			return interceptors.Failf(codes.FailedPrecondition, "error checking comments for verification: %v", err)
		}
		if !commentAllowed {
			return interceptors.Fail(codes.FailedPrecondition, "owners check requirements not met")
		}
	}

	return &triggersv1.InterceptorResponse{
		Continue: true,
	}
}

func (w *InterceptorImpl) getGithubTokenSecret(ctx context.Context, r *triggersv1.InterceptorRequest, p InterceptorParams) (string, error) {
	if p.AddChangedFiles.PersonalAccessToken == nil {
		return "", nil
	}
	if p.AddChangedFiles.PersonalAccessToken.SecretKey == "" {
		return "", errors.New("github interceptor githubToken.secretKey is empty")
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
		return results, errors.New("body is empty")
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
		if eventType == pullRequest {
			return results, errors.New("pull_request body missing 'number' field")
		}
		prNum = -1
	}

	repoSection, ok := jsonMap["repository"].(map[string]interface{})
	if !ok {
		return results, errors.New("payload body missing 'repository' field")
	}

	fullName, ok := repoSection["full_name"].(string)
	if !ok {
		return results, errors.New("payload body missing 'repository.full_name' field")
	}

	changedFiles := []string{}

	commitsSection, ok := jsonMap["commits"].([]interface{})
	if ok {
		for _, commit := range commitsSection {
			addedFiles, ok := commit.(map[string]interface{})["added"].([]interface{})
			if !ok {
				return results, errors.New("payload body missing 'commits.*.added' field")
			}

			modifiedFiles, ok := commit.(map[string]interface{})["modified"].([]interface{})
			if !ok {
				return results, errors.New("payload body missing 'commits.*.modified' field")
			}

			removedFiles, ok := commit.(map[string]interface{})["removed"].([]interface{})
			if !ok {
				return results, errors.New("payload body missing 'commits.*.removed' field")
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

func (w *InterceptorImpl) getPersonalAccessTokenSecret(ctx context.Context, r *triggersv1.InterceptorRequest, p InterceptorParams) (string, error) {
	if p.GithubOwners.PersonalAccessToken == nil {
		return "", nil
	}
	if p.GithubOwners.PersonalAccessToken.SecretKey == "" {
		return "", errors.New("github interceptor personalAccessToken.secretKey is empty")
	}
	if r.Context == nil {
		return "", errors.New("no request context passed")
	}
	ns, _ := triggersv1.ParseTriggerID(r.Context.TriggerID)
	secretToken, err := w.SecretGetter.Get(ctx, ns, p.GithubOwners.PersonalAccessToken)
	if err != nil {
		return "", err
	}
	return string(secretToken), nil
}

func okToTestFromAnOwner(ctx context.Context, payload OwnersPayloadDetails, p InterceptorParams, client *gh.Client) (bool, error) {
	if MatchRegexp(OKToTestCommentRegexp, payload.IssueCommentBody) {
		allowed, err := checkOwnershipAndMembership(ctx, payload, p, client)
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}
	return false, nil
}

func checkOwnershipAndMembership(ctx context.Context, payload OwnersPayloadDetails, p InterceptorParams, client *gh.Client) (bool, error) {
	if p.GithubOwners.CheckType == "orgMembers" || p.GithubOwners.CheckType == "all" {
		isUserMemberOrg, err := checkSenderOrgMembership(ctx, payload, client)
		if err != nil {
			return false, err
		}
		if isUserMemberOrg {
			return true, nil
		}
	}
	if p.GithubOwners.CheckType == "repoMembers" || p.GithubOwners.CheckType == "all" {
		checkSenderRepoMembership, err := checkSenderRepoMembership(ctx, payload, client)
		if err != nil {
			return false, err
		}
		if checkSenderRepoMembership {
			return true, nil
		}
	}

	ownerContent, err := getContentFromOwners(ctx, "OWNERS", payload, client)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			// no owner file, skipping
			return false, nil
		}
		return false, err
	}

	return userInOwnerFile(ownerContent, payload.Sender)
}

func checkSenderOrgMembership(ctx context.Context, payload OwnersPayloadDetails, client *gh.Client) (bool, error) {
	users, resp, err := client.Organizations.ListMembers(ctx, payload.Owner, &gh.ListMembersOptions{
		PublicOnly: true, // we can't list private member in a org
	})
	if resp != nil && resp.Response.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if err != nil {
		return false, err
	}
	for _, user := range users {
		login := *user.Login
		if login == payload.Sender {
			return true, nil
		}
	}
	return false, nil
}

func checkSenderRepoMembership(ctx context.Context, payload OwnersPayloadDetails, client *gh.Client) (bool, error) {
	users, _, err := client.Repositories.ListCollaborators(ctx, payload.Owner, payload.Repository, &gh.ListCollaboratorsOptions{})
	if err != nil {
		return false, err
	}

	for _, user := range users {
		login := *user.Login
		if login == payload.Sender {
			return true, nil
		}
	}

	return false, nil
}

func getContentFromOwners(ctx context.Context, path string, payload OwnersPayloadDetails, client *gh.Client) (string, error) {
	fileContent, directoryContent, _, err := client.Repositories.GetContents(ctx, payload.Owner, payload.Repository, path, &gh.RepositoryContentGetOptions{})

	if err != nil {
		return "", err
	}

	if directoryContent != nil {
		return "", fmt.Errorf("referenced file inside the Github Repository %s is a directory", path)
	}

	fileData, err := fileContent.GetContent()

	if err != nil {
		return "", err
	}

	return fileData, nil
}

func userInOwnerFile(ownerContent, sender string) (bool, error) {
	oc := OwnersConfig{}
	err := yaml.Unmarshal([]byte(ownerContent), &oc)
	if err != nil {
		return false, err
	}

	for _, owner := range append(oc.Approvers, oc.Reviewers...) {
		if strings.EqualFold(owner, sender) {
			return true, nil
		}
	}
	return false, nil
}

func MatchRegexp(reg, comment string) bool {
	re := regexp.MustCompile(reg)
	return string(re.Find([]byte(comment))) != ""
}

func parseBodyForOwners(body string, eventType string) (OwnersPayloadDetails, error) {
	results := OwnersPayloadDetails{}
	if body == "" {
		return results, errors.New("payload body is empty")
	}
	var jsonMap map[string]interface{}
	err := json.Unmarshal([]byte(body), &jsonMap)
	if err != nil {
		return results, err
	}

	var prNum int
	if eventType == pullRequest {
		_, ok := jsonMap["number"]
		if !ok {
			return results, errors.New("pull_request body missing 'number' field")
		}
		prNum = int(jsonMap["number"].(float64))
	} else {
		prNum = -1
	}

	var issueCommentBody string
	if eventType == "issue_comment" {
		issueSection, ok := jsonMap["issue"].(map[string]interface{})
		if !ok {
			return results, errors.New("issue_comment body missing 'issue' section")
		}
		_, ok = issueSection["number"]
		if !ok {
			return results, errors.New("'number' field missing in the issue section of issue_comment body")
		}
		prNum = int(issueSection["number"].(float64))

		issueCommentBodySection, ok := jsonMap["comment"].(map[string]interface{})
		if !ok {
			return results, errors.New("issue_comment body missing 'comment' section")
		}
		_, ok = issueCommentBodySection["body"]
		if !ok {
			return results, errors.New("'body' field missing in the comment section of issue_comment body")
		}
		issueCommentBody = issueCommentBodySection["body"].(string)
	} else {
		issueCommentBody = ""
	}

	repoSection, ok := jsonMap["repository"].(map[string]interface{})
	if !ok {
		return results, errors.New("payload body missing 'repository' field")
	}

	fullName, ok := repoSection["full_name"].(string)
	if !ok {
		return results, errors.New("payload body missing 'repository.full_name' field")
	}

	senderSection, ok := jsonMap["sender"].(map[string]interface{})
	if !ok {
		return results, errors.New("payload body missing 'sender' field")
	}
	prSender, _ := senderSection["login"].(string)

	results = OwnersPayloadDetails{
		PrNumber:         prNum,
		Sender:           prSender,
		Owner:            strings.Split(fullName, "/")[0],
		Repository:       strings.Split(fullName, "/")[1],
		IssueCommentBody: issueCommentBody,
	}

	return results, nil
}
