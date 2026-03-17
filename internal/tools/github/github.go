// Package github provides tools.Tool implementations for GitHub REST API operations.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"bruce/internal/ai"
)

// BranchItem represents a single GitHub branch.
type BranchItem struct {
	Name      string `json:"name"`
	Protected bool   `json:"protected"`
}

// BranchListResponse is the JSON response returned by the github_list_branches tool.
type BranchListResponse struct {
	Owner    string       `json:"owner"`
	Repo     string       `json:"repo"`
	Branches []BranchItem `json:"branches"`
}

// IssueItem represents a single GitHub issue.
type IssueItem struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	State  string   `json:"state"`
	URL    string   `json:"url"`
	Labels []string `json:"labels"`
}

// IssueListResponse is the JSON response returned by the github_list_issues tool.
type IssueListResponse struct {
	Owner  string      `json:"owner"`
	Repo   string      `json:"repo"`
	State  string      `json:"state"`
	Issues []IssueItem `json:"issues"`
}

// PRItem represents a single GitHub pull request.
type PRItem struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	URL    string `json:"url"`
	Head   string `json:"head"`
	Base   string `json:"base"`
}

// PRListResponse is the JSON response returned by the github_list_prs tool.
type PRListResponse struct {
	Owner string   `json:"owner"`
	Repo  string   `json:"repo"`
	State string   `json:"state"`
	PRs   []PRItem `json:"prs"`
}

// IssueCreateResponse is the JSON response returned by the github_create_issue tool.
type IssueCreateResponse struct {
	Number    int    `json:"number"`
	URL       string `json:"url"`
	CreatedAt string `json:"created_at"` // RFC3339
}

// resolveRepo returns (owner, repo) by falling back to defaults when the LLM
// omits the optional owner field from input.
func resolveRepo(input map[string]interface{}, defaultOwner, defaultRepo string) (string, string, error) {
	owner, _ := input["owner"].(string)
	if owner == "" {
		owner = defaultOwner
	}

	repo, _ := input["repo"].(string)
	if repo == "" {
		repo = defaultRepo
	}

	if owner == "" {
		return "", "", fmt.Errorf("owner is required — provide it as input or set tools.github.default_owner in config")
	}
	if repo == "" {
		return "", "", fmt.Errorf("repo is required and must be a non-empty string")
	}

	return owner, repo, nil
}

// GithubListBranchesTool implements tools.Tool for listing branches in a GitHub repo.
type GithubListBranchesTool struct {
	token        string
	defaultOwner string
	defaultRepo  string
}

// NewListBranchesTool creates a new GithubListBranchesTool.
func NewListBranchesTool(token, defaultOwner, defaultRepo string) *GithubListBranchesTool {
	return &GithubListBranchesTool{token: token, defaultOwner: defaultOwner, defaultRepo: defaultRepo}
}

// Name returns the tool name.
func (t *GithubListBranchesTool) Name() string { return "github_list_branches" }

// Definition returns the AI tool definition for github_list_branches.
func (t *GithubListBranchesTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "github_list_branches",
		Description: "List branches in a GitHub repository",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"repo": map[string]interface{}{
					"type":        "string",
					"description": "Repository name (e.g. my-repo)",
				},
				"owner": map[string]interface{}{
					"type":        "string",
					"description": "Repository owner (user or org). Falls back to configured default_owner if omitted.",
				},
				"per_page": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of branches to return (default 30)",
				},
			},
			"required": []string{"repo"},
		},
	}
}

// Execute lists branches for a GitHub repository and returns them as JSON.
func (t *GithubListBranchesTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[github] github_list_branches: Execute called")

	owner, repo, err := resolveRepo(input, t.defaultOwner, t.defaultRepo)
	if err != nil {
		return nil, err
	}

	perPage := 30
	if v, ok := input["per_page"].(float64); ok && v > 0 {
		perPage = int(v)
	}

	client := NewClient(t.token)
	branches, err := client.ListBranches(ctx, owner, repo, perPage)
	if err != nil {
		log.Printf("[github] github_list_branches: ListBranches failed: %v", err)
		return nil, fmt.Errorf("github_list_branches: %w", err)
	}

	items := make([]BranchItem, 0, len(branches))
	for _, b := range branches {
		name, _ := b["name"].(string)
		protected, _ := b["protected"].(bool)
		items = append(items, BranchItem{Name: name, Protected: protected})
	}

	resp := BranchListResponse{Owner: owner, Repo: repo, Branches: items}
	log.Printf("[github] github_list_branches: OK — count=%d", len(items))

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("github_list_branches: marshal response: %w", err)
	}
	return string(data), nil
}

// GithubListIssuesTool implements tools.Tool for listing issues in a GitHub repo.
type GithubListIssuesTool struct {
	token        string
	defaultOwner string
	defaultRepo  string
}

// NewListIssuesTool creates a new GithubListIssuesTool.
func NewListIssuesTool(token, defaultOwner, defaultRepo string) *GithubListIssuesTool {
	return &GithubListIssuesTool{token: token, defaultOwner: defaultOwner, defaultRepo: defaultRepo}
}

// Name returns the tool name.
func (t *GithubListIssuesTool) Name() string { return "github_list_issues" }

// Definition returns the AI tool definition for github_list_issues.
func (t *GithubListIssuesTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "github_list_issues",
		Description: "List issues in a GitHub repository (pull requests are excluded)",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"repo": map[string]interface{}{
					"type":        "string",
					"description": "Repository name (e.g. my-repo)",
				},
				"owner": map[string]interface{}{
					"type":        "string",
					"description": "Repository owner (user or org). Falls back to configured default_owner if omitted.",
				},
				"state": map[string]interface{}{
					"type":        "string",
					"description": "Issue state filter: open, closed, or all (default: open)",
					"enum":        []string{"open", "closed", "all"},
				},
				"per_page": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of issues to return (default 30)",
				},
			},
			"required": []string{"repo"},
		},
	}
}

// Execute lists issues for a GitHub repository and returns them as JSON.
func (t *GithubListIssuesTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[github] github_list_issues: Execute called")

	owner, repo, err := resolveRepo(input, t.defaultOwner, t.defaultRepo)
	if err != nil {
		return nil, err
	}

	state, _ := input["state"].(string)
	if state == "" {
		state = "open"
	}

	perPage := 30
	if v, ok := input["per_page"].(float64); ok && v > 0 {
		perPage = int(v)
	}

	client := NewClient(t.token)
	issues, err := client.ListIssues(ctx, owner, repo, state, perPage)
	if err != nil {
		log.Printf("[github] github_list_issues: ListIssues failed: %v", err)
		return nil, fmt.Errorf("github_list_issues: %w", err)
	}

	items := make([]IssueItem, 0, len(issues))
	for _, issue := range issues {
		// Skip pull requests (GitHub issues API returns PRs by default).
		if pr, ok := issue["pull_request"]; ok && pr != nil {
			continue
		}

		numberFloat, _ := issue["number"].(float64)
		title, _ := issue["title"].(string)
		issueState, _ := issue["state"].(string)
		url, _ := issue["html_url"].(string)

		var labelNames []string
		if labelsRaw, ok := issue["labels"].([]interface{}); ok {
			for _, l := range labelsRaw {
				if labelMap, ok := l.(map[string]interface{}); ok {
					if name, ok := labelMap["name"].(string); ok && name != "" {
						labelNames = append(labelNames, name)
					}
				}
			}
		}
		if labelNames == nil {
			labelNames = []string{}
		}

		items = append(items, IssueItem{
			Number: int(numberFloat),
			Title:  title,
			State:  issueState,
			URL:    url,
			Labels: labelNames,
		})
	}

	resp := IssueListResponse{Owner: owner, Repo: repo, State: state, Issues: items}
	log.Printf("[github] github_list_issues: OK — count=%d", len(items))

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("github_list_issues: marshal response: %w", err)
	}
	return string(data), nil
}

// GithubListPRsTool implements tools.Tool for listing pull requests in a GitHub repo.
type GithubListPRsTool struct {
	token        string
	defaultOwner string
	defaultRepo  string
}

// NewListPRsTool creates a new GithubListPRsTool.
func NewListPRsTool(token, defaultOwner, defaultRepo string) *GithubListPRsTool {
	return &GithubListPRsTool{token: token, defaultOwner: defaultOwner, defaultRepo: defaultRepo}
}

// Name returns the tool name.
func (t *GithubListPRsTool) Name() string { return "github_list_prs" }

// Definition returns the AI tool definition for github_list_prs.
func (t *GithubListPRsTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "github_list_prs",
		Description: "List pull requests in a GitHub repository",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"repo": map[string]interface{}{
					"type":        "string",
					"description": "Repository name (e.g. my-repo)",
				},
				"owner": map[string]interface{}{
					"type":        "string",
					"description": "Repository owner (user or org). Falls back to configured default_owner if omitted.",
				},
				"state": map[string]interface{}{
					"type":        "string",
					"description": "PR state filter: open, closed, or all (default: open)",
					"enum":        []string{"open", "closed", "all"},
				},
			},
			"required": []string{"repo"},
		},
	}
}

// Execute lists pull requests for a GitHub repository and returns them as JSON.
func (t *GithubListPRsTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[github] github_list_prs: Execute called")

	owner, repo, err := resolveRepo(input, t.defaultOwner, t.defaultRepo)
	if err != nil {
		return nil, err
	}

	state, _ := input["state"].(string)
	if state == "" {
		state = "open"
	}

	client := NewClient(t.token)
	prs, err := client.ListPRs(ctx, owner, repo, state)
	if err != nil {
		log.Printf("[github] github_list_prs: ListPRs failed: %v", err)
		return nil, fmt.Errorf("github_list_prs: %w", err)
	}

	items := make([]PRItem, 0, len(prs))
	for _, pr := range prs {
		numberFloat, _ := pr["number"].(float64)
		title, _ := pr["title"].(string)
		prState, _ := pr["state"].(string)
		url, _ := pr["html_url"].(string)

		var head, base string
		if headMap, ok := pr["head"].(map[string]interface{}); ok {
			head, _ = headMap["ref"].(string)
		}
		if baseMap, ok := pr["base"].(map[string]interface{}); ok {
			base, _ = baseMap["ref"].(string)
		}

		items = append(items, PRItem{
			Number: int(numberFloat),
			Title:  title,
			State:  prState,
			URL:    url,
			Head:   head,
			Base:   base,
		})
	}

	resp := PRListResponse{Owner: owner, Repo: repo, State: state, PRs: items}
	log.Printf("[github] github_list_prs: OK — count=%d", len(items))

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("github_list_prs: marshal response: %w", err)
	}
	return string(data), nil
}

// GithubCreateIssueTool implements tools.Tool for creating a new GitHub issue.
type GithubCreateIssueTool struct {
	token        string
	defaultOwner string
	defaultRepo  string
}

// NewCreateIssueTool creates a new GithubCreateIssueTool.
func NewCreateIssueTool(token, defaultOwner, defaultRepo string) *GithubCreateIssueTool {
	return &GithubCreateIssueTool{token: token, defaultOwner: defaultOwner, defaultRepo: defaultRepo}
}

// Name returns the tool name.
func (t *GithubCreateIssueTool) Name() string { return "github_create_issue" }

// Definition returns the AI tool definition for github_create_issue.
func (t *GithubCreateIssueTool) Definition() ai.ToolDefinition {
	return ai.ToolDefinition{
		Name:        "github_create_issue",
		Description: "Create a new issue in a GitHub repository",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"repo": map[string]interface{}{
					"type":        "string",
					"description": "Repository name (e.g. my-repo)",
				},
				"owner": map[string]interface{}{
					"type":        "string",
					"description": "Repository owner (user or org). Falls back to configured default_owner if omitted.",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Issue title",
				},
				"body": map[string]interface{}{
					"type":        "string",
					"description": "Optional issue body (Markdown supported)",
				},
				"labels": map[string]interface{}{
					"type":        "array",
					"description": "Optional list of label names to apply to the issue",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
			"required": []string{"repo", "title"},
		},
	}
}

// Execute creates a new GitHub issue and returns its number and URL as JSON.
func (t *GithubCreateIssueTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	log.Printf("[github] github_create_issue: Execute called")

	owner, repo, err := resolveRepo(input, t.defaultOwner, t.defaultRepo)
	if err != nil {
		return nil, err
	}

	title, ok := input["title"].(string)
	if !ok || title == "" {
		return nil, fmt.Errorf("title is required and must be a non-empty string")
	}

	body, _ := input["body"].(string)

	var labels []string
	if raw, ok := input["labels"].([]interface{}); ok {
		for _, l := range raw {
			if s, ok := l.(string); ok && s != "" {
				labels = append(labels, s)
			}
		}
	}

	log.Printf("[github] github_create_issue: owner=%q repo=%q title=%q", owner, repo, title)

	client := NewClient(t.token)
	number, issueURL, err := client.CreateIssue(ctx, owner, repo, title, body, labels)
	if err != nil {
		log.Printf("[github] github_create_issue: CreateIssue failed: %v", err)
		return nil, fmt.Errorf("github_create_issue: %w", err)
	}

	resp := IssueCreateResponse{
		Number:    number,
		URL:       issueURL,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	log.Printf("[github] github_create_issue: OK — number=%d url=%q", number, issueURL)

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("github_create_issue: marshal response: %w", err)
	}
	return string(data), nil
}
