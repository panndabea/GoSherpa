package git

import (
	"fmt"
	"os/exec"
	"strings"
)

const (
	BaseSourceExplicit = "explicit"
	BaseSourceDetected = "detected"
	BaseSourceFallback = "fallback"
)

type BaseDetection struct {
	Selected   string          `json:"selected"`
	Source     string          `json:"source"`
	Candidates []BaseCandidate `json:"candidates"`
	Warnings   []string        `json:"warnings"`
}

type BaseCandidate struct {
	Ref      string `json:"ref"`
	Resolved bool   `json:"resolved"`
	Commit   string `json:"commit,omitempty"`
	Message  string `json:"message,omitempty"`
}

var defaultBaseCandidates = []string{"origin/main", "main", "origin/master", "master", "HEAD"}

func DetectBase(root string, explicitRef string) (BaseDetection, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return BaseDetection{}, fmt.Errorf("repository root is empty")
	}

	explicitRef = strings.TrimSpace(explicitRef)
	if explicitRef != "" {
		candidate := verifyBaseCandidate(root, explicitRef)
		result := BaseDetection{
			Selected:   explicitRef,
			Source:     BaseSourceExplicit,
			Candidates: []BaseCandidate{candidate},
		}
		if !candidate.Resolved {
			return result, fmt.Errorf("base ref %q could not be resolved locally: %s", explicitRef, candidate.Message)
		}
		return result, nil
	}

	if !insideGitWorkTree(root) {
		return BaseDetection{
			Selected: BaseSourceFallbackRef(),
			Source:   BaseSourceFallback,
			Warnings: []string{"repository is not a git work tree; using HEAD as the fallback base. Run gosherpa init --base <ref> to set a stable base."},
		}, nil
	}

	result := BaseDetection{
		Source:     BaseSourceFallback,
		Candidates: make([]BaseCandidate, 0, len(defaultBaseCandidates)),
	}
	for _, ref := range defaultBaseCandidates {
		candidate := verifyBaseCandidate(root, ref)
		result.Candidates = append(result.Candidates, candidate)
		if candidate.Resolved {
			result.Selected = ref
			result.Source = BaseSourceDetected
			return result, nil
		}
	}

	result.Selected = BaseSourceFallbackRef()
	result.Warnings = []string{"no local base ref resolved from origin/main, main, origin/master, master, or HEAD; using HEAD as the fallback base. Run gosherpa init --base <ref> to set a stable base."}
	return result, nil
}

func BaseSourceFallbackRef() string {
	return "HEAD"
}

func insideGitWorkTree(root string) bool {
	output, err := exec.Command("git", "-C", root, "rev-parse", "--is-inside-work-tree").CombinedOutput()
	return err == nil && strings.TrimSpace(string(output)) == "true"
}

func verifyBaseCandidate(root string, ref string) BaseCandidate {
	output, err := exec.Command("git", "-C", root, "rev-parse", "--verify", ref).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return BaseCandidate{
			Ref:      ref,
			Resolved: false,
			Message:  message,
		}
	}

	return BaseCandidate{
		Ref:      ref,
		Resolved: true,
		Commit:   strings.TrimSpace(string(output)),
	}
}
