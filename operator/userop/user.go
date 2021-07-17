package userop

import (
	"bufio"
	"os"
	"os/user"
	"strings"
)

const userFile = "/etc/passwd"

type User struct {
	*user.User
	Shell string
}

func Lookup(username string) (*User, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return nil, err
	}

	shell, err := lookupShell(username)
	return &User{
		User:  u,
		Shell: shell,
	}, err
}

func (u *User) ToMap() map[string]interface{} {
	if u == nil {
		return nil
	}
	return map[string]interface{}{
		"uid":      u.Uid,
		"gid":      u.Gid,
		"username": u.Username,
		"name":     u.Name,
		"home":     u.HomeDir,
		"shell":    u.Shell,
	}
}

func lookupShell(username string) (string, error) {
	f, err := os.Open(userFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	var matchLine string
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, username+":") {
			matchLine = line
			break
		}
	}
	if err := s.Err(); err != nil {
		return "", err
	}

	if matchLine == "" {
		return "", user.UnknownUserError(username)
	}

	parts := strings.SplitN(matchLine, ":", 7)
	if len(parts) < 6 || parts[0] == "" ||
		parts[0][0] == '+' || parts[0][0] == '-' {
		return "", nil
	}
	return parts[6], nil
}
