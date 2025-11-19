package version

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Version represents a semantic version
type Version struct {
	Major       int
	Minor       int
	Maintenance string
	GitDescribe string
	IsDirty     bool
}

// String returns the version as a string
func (v *Version) String() string {
	return fmt.Sprintf("%d.%d.%s", v.Major, v.Minor, v.Maintenance)
}

// GetFromGit gets the version from git tags
func GetFromGit(dir string) (*Version, error) {
	// Get git describe output
	cmd := exec.Command("git", "describe", "--tags", "--match", "v*.*.*")
	cmd.Dir = dir
	output, err := cmd.Output()

	gitDescribe := "v0.1.0"
	if err == nil {
		gitDescribe = strings.TrimSpace(string(output))
	}

	// Parse git describe output
	// Format: v0.1.0 or v0.1.0-5-g1a2b3c4
	re := regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)(?:-(\d+)-g([0-9a-f]+))?$`)
	matches := re.FindStringSubmatch(gitDescribe)

	var major, minor int
	var maintenance string

	if matches != nil {
		fmt.Sscanf(matches[1], "%d", &major)
		fmt.Sscanf(matches[2], "%d", &minor)
		maintenance = matches[3]

		// If there are commits after the tag, append commit count
		if matches[4] != "" {
			maintenance = fmt.Sprintf("%s-%s", maintenance, matches[4])
		}
	} else {
		// Fallback
		major = 0
		minor = 1
		maintenance = "0"
	}

	// Check for uncommitted changes
	isDirty := false
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	output, err = cmd.Output()
	if err == nil && len(strings.TrimSpace(string(output))) > 0 {
		isDirty = true
		timestamp := time.Now().Format("01021504")
		maintenance = fmt.Sprintf("%s-%s", maintenance, timestamp)
	}

	return &Version{
		Major:       major,
		Minor:       minor,
		Maintenance: maintenance,
		GitDescribe: gitDescribe,
		IsDirty:     isDirty,
	}, nil
}

// IsGitRepo checks if the directory is a git repository
func IsGitRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = dir
	return cmd.Run() == nil
}
