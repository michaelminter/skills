package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const usage = `Usage: skills <command> [arguments]

Commands:
  sync                  Link repository skills into ~/.agents and ~/.claude
  reset                 Remove repository skills from ~/.agents and ~/.claude
  add <skill>           Create a skill directory and SKILL.md
  delete <skill>        Delete a skill directory from this repository
  rename <old> <new>    Rename a skill and re-sync installed links
  help                  Show this help
`

var validSkillName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
var frontmatterName = regexp.MustCompile(`(?m)^name:[^\r\n]*`)

type cli struct {
	root   string
	home   string
	stdout io.Writer
}

func main() {
	if len(os.Args) == 1 {
		fmt.Print(usage)
		return
	}
	if os.Args[1] == "help" || os.Args[1] == "-h" || os.Args[1] == "--help" {
		app := cli{stdout: os.Stdout}
		if err := app.run(os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		return
	}
	if os.Args[1] != "sync" && os.Args[1] != "reset" && os.Args[1] != "add" && os.Args[1] != "delete" && os.Args[1] != "rename" {
		app := cli{stdout: os.Stdout}
		fmt.Fprintln(os.Stderr, "Error:", app.run(os.Args[1:]))
		os.Exit(1)
	}

	root, err := findRepositoryRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: determine home directory:", err)
		os.Exit(1)
	}

	app := cli{root: root, home: home, stdout: os.Stdout}
	if err := app.run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func (c cli) run(args []string) error {
	if len(args) == 0 {
		fmt.Fprint(c.stdout, usage)
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		if len(args) != 1 {
			return fmt.Errorf("help does not accept arguments")
		}
		fmt.Fprint(c.stdout, usage)
		return nil
	case "sync":
		if len(args) != 1 {
			return fmt.Errorf("usage: skills sync")
		}
		return c.sync()
	case "reset":
		if len(args) != 1 {
			return fmt.Errorf("usage: skills reset")
		}
		return c.reset()
	case "add":
		if len(args) != 2 {
			return fmt.Errorf("usage: skills add <skill>")
		}
		return c.add(args[1])
	case "delete":
		if len(args) != 2 {
			return fmt.Errorf("usage: skills delete <skill>")
		}
		return c.delete(args[1])
	case "rename":
		if len(args) != 3 {
			return fmt.Errorf("usage: skills rename <old> <new>")
		}
		return c.rename(args[1], args[2])
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], usage)
	}
}

func (c cli) sync() error {
	skills, err := c.skills()
	if err != nil {
		return err
	}
	if len(skills) == 0 {
		return fmt.Errorf("no skill directories containing SKILL.md were found in %s", c.skillsDir())
	}

	for _, destination := range c.destinations() {
		if err := os.MkdirAll(destination, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", destination, err)
		}
	}

	for _, skill := range skills {
		source := filepath.Join(c.skillsDir(), skill)
		for _, destination := range c.destinations() {
			target := filepath.Join(destination, skill)
			if err := os.RemoveAll(target); err != nil {
				return fmt.Errorf("remove existing %s: %w", target, err)
			}
			if err := os.Symlink(source, target); err != nil {
				return fmt.Errorf("link %s to %s: %w", target, source, err)
			}
			fmt.Fprintf(c.stdout, "Linked %s to %s\n", skill, target)
		}
	}

	fmt.Fprintf(c.stdout, "Linked %d skill(s) to both destinations.\n", len(skills))
	return nil
}

func (c cli) reset() error {
	skills, err := c.skills()
	if err != nil {
		return err
	}

	removed := 0
	for _, destination := range c.destinations() {
		installedSkills, err := c.installedSkills(destination, skills)
		if err != nil {
			return err
		}
		for _, skill := range installedSkills {
			target := filepath.Join(destination, skill)
			exists, err := pathExists(target)
			if err != nil {
				return fmt.Errorf("inspect %s: %w", target, err)
			}
			if !exists {
				continue
			}
			if err := os.RemoveAll(target); err != nil {
				return fmt.Errorf("remove %s: %w", target, err)
			}
			removed++
			fmt.Fprintf(c.stdout, "Removed %s\n", target)
		}
	}

	fmt.Fprintf(c.stdout, "Removed %d installed skill(s).\n", removed)
	return nil
}

func (c cli) installedSkills(destination string, repositorySkills []string) ([]string, error) {
	managed := make(map[string]bool, len(repositorySkills))
	for _, skill := range repositorySkills {
		managed[skill] = true
	}

	entries, err := os.ReadDir(destination)
	if errors.Is(err, os.ErrNotExist) {
		return repositorySkills, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read destination %s: %w", destination, err)
	}

	repositorySkillsDir := filepath.Clean(c.skillsDir())
	for _, entry := range entries {
		target := filepath.Join(destination, entry.Name())
		info, err := os.Lstat(target)
		if err != nil {
			return nil, fmt.Errorf("inspect %s: %w", target, err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}

		source, err := os.Readlink(target)
		if err != nil {
			return nil, fmt.Errorf("read link %s: %w", target, err)
		}
		if !filepath.IsAbs(source) {
			source = filepath.Join(destination, source)
		}
		source = filepath.Clean(source)
		if filepath.Dir(source) == repositorySkillsDir && filepath.Base(source) == entry.Name() {
			managed[entry.Name()] = true
		}
	}

	result := make([]string, 0, len(managed))
	for skill := range managed {
		result = append(result, skill)
	}
	sort.Strings(result)
	return result, nil
}

func (c cli) add(name string) error {
	if err := validateSkillName(name); err != nil {
		return err
	}

	if err := os.MkdirAll(c.skillsDir(), 0o755); err != nil {
		return fmt.Errorf("create skills directory: %w", err)
	}

	dir := filepath.Join(c.skillsDir(), name)
	if err := os.Mkdir(dir, 0o755); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("skill %q already exists", name)
		}
		return fmt.Errorf("create skill directory: %w", err)
	}

	contents := fmt.Sprintf("---\nname: %s\ndescription: \"TODO: Describe what this skill does and when to use it.\"\n---\n\n# %s\n", name, name)
	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		_ = os.RemoveAll(dir)
		return fmt.Errorf("create SKILL.md: %w", err)
	}

	fmt.Fprintf(c.stdout, "Created %s\n", path)
	return nil
}

func (c cli) delete(name string) error {
	if err := validateSkillName(name); err != nil {
		return err
	}

	dir := filepath.Join(c.skillsDir(), name)
	info, err := os.Stat(filepath.Join(dir, "SKILL.md"))
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("skill %q does not exist", name)
	}
	if err != nil {
		return fmt.Errorf("inspect skill %q: %w", name, err)
	}
	if info.IsDir() {
		return fmt.Errorf("skill %q has a SKILL.md directory instead of a file", name)
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("delete skill %q: %w", name, err)
	}
	fmt.Fprintf(c.stdout, "Deleted %s\n", dir)
	return nil
}

func (c cli) rename(oldName, newName string) error {
	if err := validateSkillName(oldName); err != nil {
		return err
	}
	if err := validateSkillName(newName); err != nil {
		return err
	}
	if oldName == newName {
		return fmt.Errorf("old and new skill names must be different")
	}

	oldDir := filepath.Join(c.skillsDir(), oldName)
	oldSkillFile := filepath.Join(oldDir, "SKILL.md")
	info, err := os.Stat(oldSkillFile)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("skill %q does not exist", oldName)
	}
	if err != nil {
		return fmt.Errorf("inspect skill %q: %w", oldName, err)
	}
	if info.IsDir() {
		return fmt.Errorf("skill %q has a SKILL.md directory instead of a file", oldName)
	}

	newDir := filepath.Join(c.skillsDir(), newName)
	exists, err := pathExists(newDir)
	if err != nil {
		return fmt.Errorf("inspect new skill path: %w", err)
	}
	if exists {
		return fmt.Errorf("skill %q already exists", newName)
	}

	contents, err := os.ReadFile(oldSkillFile)
	if err != nil {
		return fmt.Errorf("read skill %q: %w", oldName, err)
	}
	updatedContents, err := renameFrontmatterSkill(contents, newName)
	if err != nil {
		return fmt.Errorf("rename skill %q: %w", oldName, err)
	}

	if err := os.Rename(oldDir, newDir); err != nil {
		return fmt.Errorf("rename skill directory: %w", err)
	}
	newSkillFile := filepath.Join(newDir, "SKILL.md")
	if err := os.WriteFile(newSkillFile, updatedContents, info.Mode().Perm()); err != nil {
		if rollbackErr := os.Rename(newDir, oldDir); rollbackErr != nil {
			return fmt.Errorf("update renamed SKILL.md: %w (rollback failed: %v)", err, rollbackErr)
		}
		return fmt.Errorf("update renamed SKILL.md: %w", err)
	}

	fmt.Fprintf(c.stdout, "Renamed %s to %s\n", oldName, newName)
	for _, destination := range c.destinations() {
		target := filepath.Join(destination, oldName)
		exists, err := pathExists(target)
		if err != nil {
			return fmt.Errorf("inspect old installed skill %s: %w", target, err)
		}
		if !exists {
			continue
		}
		if err := os.RemoveAll(target); err != nil {
			return fmt.Errorf("remove old installed skill %s: %w", target, err)
		}
		fmt.Fprintf(c.stdout, "Removed %s\n", target)
	}

	return c.sync()
}

func renameFrontmatterSkill(contents []byte, newName string) ([]byte, error) {
	text := string(contents)
	if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
		return nil, fmt.Errorf("SKILL.md does not start with YAML frontmatter")
	}

	frontmatterEnd := strings.Index(text[3:], "\n---")
	if frontmatterEnd < 0 {
		return nil, fmt.Errorf("SKILL.md has no closing YAML frontmatter delimiter")
	}
	frontmatterEnd += 3
	nameLocation := frontmatterName.FindStringIndex(text[:frontmatterEnd])
	if nameLocation == nil {
		return nil, fmt.Errorf("SKILL.md frontmatter has no name field")
	}

	updated := text[:nameLocation[0]] + "name: " + newName + text[nameLocation[1]:]
	return []byte(updated), nil
}

func (c cli) skills() ([]string, error) {
	entries, err := os.ReadDir(c.skillsDir())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read skills directory: %w", err)
	}

	var skills []string
	for _, entry := range entries {
		dir := filepath.Join(c.skillsDir(), entry.Name())
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		skillFile, err := os.Stat(filepath.Join(dir, "SKILL.md"))
		if err == nil && !skillFile.IsDir() {
			skills = append(skills, entry.Name())
		}
	}
	sort.Strings(skills)
	return skills, nil
}

func (c cli) skillsDir() string {
	return filepath.Join(c.root, "skills")
}

func (c cli) destinations() []string {
	return []string{
		filepath.Join(c.home, ".agents", "skills"),
		filepath.Join(c.home, ".claude", "skills"),
	}
}

func validateSkillName(name string) error {
	if !validSkillName.MatchString(name) || name == "." || name == ".." {
		return fmt.Errorf("invalid skill name %q; use letters, numbers, dots, underscores, and hyphens", name)
	}
	return nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func findRepositoryRoot() (string, error) {
	var starts []string
	if cwd, err := os.Getwd(); err == nil {
		starts = append(starts, cwd)
	}
	if executable, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(executable))
	}

	for _, start := range starts {
		for dir := start; ; dir = filepath.Dir(dir) {
			contents, err := os.ReadFile(filepath.Join(dir, "go.mod"))
			if err == nil && strings.Contains(string(contents), "module github.com/michaelminter/skills") {
				return filepath.Abs(dir)
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
	}

	return "", fmt.Errorf("could not find the skills repository; run this command from the repository or keep the binary inside it")
}
