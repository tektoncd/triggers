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

const (
	apiPublicURL          = "https://api.github.com/"
	OKToTestCommentRegexp = `(^|\n)\/ok-to-test(\r\n|\r|\n|$)`
)

var _ triggersv1.InterceptorInterface = (*Interceptor)(nil)

var secretToken []byte
var err error

// ErrInvalidContentType is returned when the content-type is not a JSON body.
var ErrInvalidContentType = errors.New("form parameter encoding not supported, please change the hook to send JSON payloads")

type Interceptor struct {
	SecretGetter interceptors.SecretGetter
}

type OwnersPayloadDetails struct {
	PrNumber   int
	Sender     string
	Owner      string
	Repository string
}

type OwnersConfig struct {
	Approvers []string `json:"approvers,omitempty"`
	Reviewers []string `json:"reviewers,omitempty"`
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

	// Check if the event type is in the allow-list
	if p.EventTypes != nil {
		actualEvent := headers.Get("X-GitHub-Event")
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
		secretToken, err = w.SecretGetter.Get(ctx, ns, p.SecretRef)
		if err != nil {
			return interceptors.Failf(codes.FailedPrecondition, "error getting secret: %v", err)
		}

		if err := gh.ValidateSignature(header, []byte(r.Body), secretToken); err != nil {
			return interceptors.Fail(codes.FailedPrecondition, err.Error())
		}
	}

	var ownersEventTypes = []string{"pull_request", "issue_comment"}

	// For event types pull_request, issue_comment check github owners approval
	if p.EventTypes != nil {
		actualEvent := headers.Get("X-GitHub-Event")
		ownerCheckAllowed := false
		for _, allowedEvent := range ownersEventTypes {
			if actualEvent == allowedEvent {
				ownerCheckAllowed = true
				break
			}
		}
		if ownerCheckAllowed {
			client := makeClient(ctx, headers.Get("X-Github-Enterprise-Host"), string(secretToken))
			payload, err := parseBody(r.Body, actualEvent)
			if err != nil {
				return interceptors.Failf(codes.FailedPrecondition, "error parsing body: %v", err)
			}
			allowed, err := checkOwnershipAndMembership(ctx, payload, p, client)
			if err != nil {
				return interceptors.Failf(codes.FailedPrecondition, "error checking owner verification: %v", err)
			}

			if allowed {
				return &triggersv1.InterceptorResponse{
					Continue: true,
				}
			}

			commentAllowed, err := okToTestFromAnOwner(ctx, payload, p, client)
			if err != nil {
				return interceptors.Failf(codes.FailedPrecondition, "error checking comments for verification: %v", err)
			}
			if !commentAllowed {
				return &triggersv1.InterceptorResponse{
					Continue: false,
				}
			}
		}
	}

	return &triggersv1.InterceptorResponse{
		Continue: true,
	}
}

func okToTestFromAnOwner(ctx context.Context, payload OwnersPayloadDetails, p triggersv1.GitHubInterceptor, client *gh.Client) (bool, error) {

	comments, err := getStringPullRequestComment(ctx, payload, client)
	if err != nil {
		return false, err
	}

	for _, comment := range comments {
		payload.Sender = comment.User.GetLogin()
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

func checkOwnershipAndMembership(ctx context.Context, payload OwnersPayloadDetails, p triggersv1.GitHubInterceptor, client *gh.Client) (bool, error) {
	if p.EnableOrgMemberCheck {
		isUserMemberRepo, err := checkSenderOrgMembership(ctx, payload, client)
		if err != nil {
			return false, err
		}
		if isUserMemberRepo {
			return true, nil
		}
	}
	if p.EnableRepoMemberCheck {
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
		PublicOnly: true, //we can't list private member in a org
	})
	if resp != nil && resp.Response.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if err != nil {
		return false, err
	}
	for _, user := range users {
		if user.GetLogin() == payload.Sender {
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
		if user.GetLogin() == payload.Sender {
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

func getStringPullRequestComment(ctx context.Context, payload OwnersPayloadDetails, client *gh.Client) ([]*gh.PullRequestComment, error) {
	var ret []*gh.PullRequestComment
	comments, _, err := client.PullRequests.ListComments(ctx, payload.Owner, payload.Repository, payload.PrNumber, &gh.PullRequestListCommentsOptions{})
	if err != nil {
		return ret, err
	}
	for _, comment := range comments {
		if MatchRegexp(OKToTestCommentRegexp, string(*comment.Body)) {
			ret = append(ret, comment)
		}
	}
	return ret, nil
}

func MatchRegexp(reg, comment string) bool {
	re := regexp.MustCompile(reg)
	return string(re.Find([]byte(comment))) != ""
}

// type cfgKey struct{}

func makeClient(ctx context.Context, apiURL string, token string) *gh.Client {
	var client *gh.Client
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	tokenClient := oauth2.NewClient(ctx, tokenSource)
	if apiURL != "" {
		apiURL = "https://" + apiURL
	}
	if apiURL != "" && apiURL != apiPublicURL {
		client, _ = gh.NewEnterpriseClient(apiURL, apiURL, tokenClient)
	} else {
		client = gh.NewClient(tokenClient)
		// testUrl := ctx.Value(cfgKey{}).(string)
		// s := strings.Split(testUrl, ":")
		// client.BaseURL = &url.URL{
		// 	Scheme: s[0],
		// 	Host:   s[1],
		// }
	}
	return client
}

func parseBody(body string, eventType string) (OwnersPayloadDetails, error) {
	results := OwnersPayloadDetails{}
	if body == "" {
		return results, fmt.Errorf("body is empty")
	}
	var jsonMap map[string]interface{}
	err := json.Unmarshal([]byte(body), &jsonMap)
	if err != nil {
		return results, err
	}

	var prNum int
	if eventType == "pull_request" {
		_, ok := jsonMap["number"]
		if !ok && eventType == "pull_request" {
			return results, fmt.Errorf("pull_request body missing 'number' field")
		} else if eventType == "pull_request" {
			prNum = int(jsonMap["number"].(float64))
		} else {
			prNum = -1
		}
	}

	if eventType == "issue_comment" {
		issueSection, ok := jsonMap["issue"].(map[string]interface{})
		if !ok {
			return results, fmt.Errorf("issue_comment body missing 'issue' section")
		}
		_, ok = issueSection["number"]
		if !ok {
			return results, fmt.Errorf("'number' field missing in the issue section of issue_comment body")
		}
		prNum = int(issueSection["number"].(float64))
	}

	repoSection, ok := jsonMap["repository"].(map[string]interface{})
	if !ok {
		return results, fmt.Errorf("payload body missing 'repository' field")
	}

	fullName, ok := repoSection["full_name"].(string)
	if !ok {
		return results, fmt.Errorf("payload body missing 'repository.full_name' field")
	}

	senderSection, ok := jsonMap["sender"].(map[string]interface{})
	if !ok {
		return results, fmt.Errorf("payload body missing 'sender' field")
	}
	prSender, _ := senderSection["login"].(string)

	results = OwnersPayloadDetails{
		PrNumber:   prNum,
		Sender:     prSender,
		Owner:      strings.Split(fullName, "/")[0],
		Repository: strings.Split(fullName, "/")[1],
	}

	return results, nil
}
