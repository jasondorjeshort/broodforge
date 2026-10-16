// SPDX-License-Identifier: GPL-3.0-or-later

package git

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GitHubAPI is the GitHub REST API's root.
const GitHubAPI = "https://api.github.com"

// PullRequestBody is the request's JSON body (GitHub's create-a-pull-request).
type PullRequestBody struct {
	Title string `json:"title"`
	Head  string `json:"head"`
	Base  string `json:"base"`
	Body  string `json:"body"`
}

// PullRequest is one request as `--dry-run` prints it: the endpoint, the credential by id
// (never its value) and the body.
type PullRequest struct {
	URL        string          `json:"url"`
	Credential string          `json:"credential"`
	Body       PullRequestBody `json:"body"`
}

// PullRequester opens a pull request with the project's git credential in that one call
// (03-architecture.md, End of session) and returns the request's web URL.
type PullRequester interface {
	Open(ctx context.Context, pr PullRequest, token []byte) (string, error)
}

// GitHub is the github forge: one POST to <API>/repos/<owner>/<repo>/pulls.
type GitHub struct {
	Client *http.Client // nil: a client with a 30-second timeout
}

var _ PullRequester = GitHub{}

// githubRemote matches https://github.com/<owner>/<repo>[.git] and git@github.com:<owner>/<repo>[.git].
var githubRemote = regexp.MustCompile(`^(?:https://(?:[^@/]+@)?github\.com/|git@github\.com:|ssh://git@github\.com/)([A-Za-z0-9_.-]+)/([A-Za-z0-9_.-]+?)(?:\.git)?/?$`)

// GitHubRepo is the owner and repository of a GitHub remote URL.
func GitHubRepo(remoteURL string) (owner, repo string, ok bool) {
	m := githubRemote.FindStringSubmatch(strings.TrimSpace(remoteURL))
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}

// PullsURL is the endpoint that opens a pull request on owner/repo under api.
func PullsURL(api, owner, repo string) string {
	return strings.TrimSuffix(api, "/") + "/repos/" + owner + "/" + repo + "/pulls"
}

// Open posts the request. A status other than 201 is handoff-failed-pull-request with the
// status and GitHub's message.
func (g GitHub) Open(ctx context.Context, pr PullRequest, token []byte) (string, error) {
	body, err := json.Marshal(pr.Body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, pr.URL, bytes.NewReader(body))
	if err != nil {
		return "", &Error{Code: CodeFailedPullRequest, Message: err.Error()}
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if len(token) > 0 {
		req.Header.Set("Authorization", "Bearer "+string(token))
	}
	client := g.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", &Error{Code: CodeFailedPullRequest, Message: err.Error()}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	var answer struct {
		HTMLURL string `json:"html_url"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(raw, &answer)
	if resp.StatusCode != http.StatusCreated {
		msg := "HTTP " + strconv.Itoa(resp.StatusCode)
		if answer.Message != "" {
			msg += ": " + answer.Message
		}
		return "", &Error{Code: CodeFailedPullRequest, Message: msg}
	}
	if answer.HTMLURL == "" {
		return "", &Error{Code: CodeFailedPullRequest, Message: "HTTP 201 without html_url"}
	}
	return answer.HTMLURL, nil
}
