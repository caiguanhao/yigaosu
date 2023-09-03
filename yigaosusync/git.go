package main

import (
	"log"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	xssh "golang.org/x/crypto/ssh"
)

type (
	gitClient struct {
		Remote    string
		RemoteUrl string
		Branch    string
		LocalDir  string
		SshKey    string
	}

	addFilesOpts struct {
		UserName  string
		UserEmail string
		AddFiles  func(worktree *git.Worktree) (string, error)
	}
)

func (c *gitClient) AddFiles(opts addFilesOpts) error {
	publicKey, err := ssh.NewPublicKeys("git", []byte(c.SshKey), "")
	if err != nil {
		return err
	}
	publicKey.HostKeyCallback = xssh.InsecureIgnoreHostKey()
	log.Println("git cloning from", c.RemoteUrl)
	r, err := git.PlainClone(c.LocalDir, false, &git.CloneOptions{
		URL:  c.RemoteUrl,
		Auth: publicKey,
	})
	if err == transport.ErrEmptyRemoteRepository {
		log.Println("init", c.LocalDir)
		r, err = git.PlainInit(c.LocalDir, false)
		if err == nil {
			_, err = r.CreateRemote(&config.RemoteConfig{
				Name: c.Remote,
				URLs: []string{c.RemoteUrl},
			})
		}
	}
	if err == git.ErrRepositoryAlreadyExists {
		r, err = git.PlainOpen(c.LocalDir)
	}
	if err != nil {
		return err
	}
	w, err := r.Worktree()
	if err != nil {
		return err
	}
	log.Println("fetching", c.Remote)
	err = r.Fetch(&git.FetchOptions{
		RemoteName: c.Remote,
		Auth:       publicKey,
		Force:      true,
	})
	if err == transport.ErrEmptyRemoteRepository {
		err = nil
	} else {
		if err != nil && err != git.NoErrAlreadyUpToDate {
			return err
		}
		ref, e := r.Reference(plumbing.NewRemoteReferenceName(c.Remote, c.Branch), true)
		if e != nil {
			return e
		}
		err = w.Checkout(&git.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(c.Branch),
			Force:  true,
		})
		if err != nil {
			return err
		}
		err = w.Reset(&git.ResetOptions{
			Mode:   git.HardReset,
			Commit: ref.Hash(),
		})
	}
	if err != nil {
		return err
	}
	var commitMsg string
	commitMsg, err = opts.AddFiles(w)
	if err != nil {
		return err
	}
	s, err := w.Status()
	if err != nil {
		return err
	}
	if len(s) == 0 {
		log.Println("no changes")
		return nil
	}
	commit, err := w.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  opts.UserName,
			Email: opts.UserEmail,
			When:  time.Now(),
		},
	})
	log.Println("adding commit", commit.String()[:8], commitMsg)
	if err != nil {
		return err
	}
	log.Println("pushing")
	err = r.Push(&git.PushOptions{
		Auth: publicKey,
	})
	if err == nil {
		log.Println("pushed")
	}
	return err
}

func (c *gitClient) ForcePush() error {
	r, err := git.PlainOpen(c.LocalDir)
	if err != nil {
		return err
	}
	publicKey, err := ssh.NewPublicKeys("git", []byte(c.SshKey), "")
	if err != nil {
		return err
	}
	publicKey.HostKeyCallback = xssh.InsecureIgnoreHostKey()
	err = r.Push(&git.PushOptions{
		Auth:  publicKey,
		Force: true,
	})
	if err == nil {
		log.Println("pushed")
	}
	return err
}
