package worktree

import (
	"fmt"

	"github.com/kyriacos/go-gwt/internal/ui"
)

const upstreamRemote = "origin"

// alignBranchUpstream configures each branch to track origin/<branch> so a later
// `git push` targets the correct remote branch. It does not push.
func (s *Service) alignBranchUpstream(branch string) {
	if branch == "" {
		return
	}
	if err := s.doAlignBranchUpstream(branch); err != nil {
		ui.Warn("upstream: %v", err)
	}
}

func (s *Service) doAlignBranchUpstream(branch string) error {
	def, err := s.Repo.DefaultBranch()
	if err != nil {
		return fmt.Errorf("default branch: %w", err)
	}

	if branch == def {
		return s.alignDefaultBranchUpstream(branch, def)
	}
	return s.alignFeatureBranchUpstream(branch)
}

func (s *Service) alignDefaultBranchUpstream(branch, def string) error {
	exists, err := s.Repo.RemoteBranchExists(upstreamRemote, def)
	if err != nil {
		return err
	}
	if exists {
		return s.setUpstreamIfNeeded(branch, upstreamRemote, def)
	}
	return s.unsetUpstreamIfConfigured(branch)
}

func (s *Service) alignFeatureBranchUpstream(branch string) error {
	return s.setUpstreamIfNeeded(branch, upstreamRemote, branch)
}

func (s *Service) setUpstreamIfNeeded(branch, remote, upstreamBranch string) error {
	curRemote, curBranch, configured, err := s.Repo.BranchUpstream(branch)
	if err != nil {
		return err
	}
	if configured && curRemote == remote && curBranch == upstreamBranch {
		return nil
	}
	return s.Repo.SetUpstream(branch, remote, upstreamBranch)
}

func (s *Service) unsetUpstreamIfConfigured(branch string) error {
	_, _, configured, err := s.Repo.BranchUpstream(branch)
	if err != nil {
		return err
	}
	if !configured {
		return nil
	}
	return s.Repo.UnsetUpstream(branch)
}
