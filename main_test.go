package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testCLI(t *testing.T) (cli, *bytes.Buffer) {
	t.Helper()
	output := &bytes.Buffer{}
	return cli{root: t.TempDir(), home: t.TempDir(), stdout: output}, output
}

func TestAddAndDelete(t *testing.T) {
	app, output := testCLI(t)

	if err := app.run([]string{"add", "example-skill"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(app.skillsDir(), "example-skill", "SKILL.md")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "name: example-skill") {
		t.Fatalf("SKILL.md did not contain the skill name:\n%s", contents)
	}

	if err := app.run([]string{"delete", "example-skill"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatalf("skill directory still exists: %v", err)
	}
	if !strings.Contains(output.String(), "Created") || !strings.Contains(output.String(), "Deleted") {
		t.Fatalf("unexpected output: %s", output.String())
	}
}

func TestRejectsUnsafeSkillNames(t *testing.T) {
	app, _ := testCLI(t)
	for _, name := range []string{"../outside", "two words", ".", ""} {
		if err := app.run([]string{"add", name}); err == nil {
			t.Errorf("add accepted unsafe name %q", name)
		}
	}
}

func TestRenameUpdatesSkillAndInstalledLinks(t *testing.T) {
	app, output := testCLI(t)
	if err := app.run([]string{"add", "old-skill"}); err != nil {
		t.Fatal(err)
	}
	oldDir := filepath.Join(app.skillsDir(), "old-skill")
	if err := os.WriteFile(filepath.Join(oldDir, "support.txt"), []byte("support"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := app.run([]string{"sync"}); err != nil {
		t.Fatal(err)
	}

	if err := app.run([]string{"rename", "old-skill", "new-skill"}); err != nil {
		t.Fatal(err)
	}
	newDir := filepath.Join(app.skillsDir(), "new-skill")
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Fatalf("old skill directory still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(newDir, "support.txt")); err != nil {
		t.Fatalf("support file was not moved: %v", err)
	}
	contents, err := os.ReadFile(filepath.Join(newDir, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "name: new-skill") || strings.Contains(string(contents), "name: old-skill") {
		t.Fatalf("SKILL.md name was not updated:\n%s", contents)
	}

	for _, destination := range app.destinations() {
		if _, err := os.Lstat(filepath.Join(destination, "old-skill")); !os.IsNotExist(err) {
			t.Fatalf("old installed reference still exists: %v", err)
		}
		newLink := filepath.Join(destination, "new-skill")
		linked, err := os.Readlink(newLink)
		if err != nil {
			t.Fatalf("new installed reference is not a link: %v", err)
		}
		if linked != newDir {
			t.Fatalf("new link points to %q, want %q", linked, newDir)
		}
	}
	if !strings.Contains(output.String(), "Renamed old-skill to new-skill") {
		t.Fatalf("unexpected output: %s", output.String())
	}
}

func TestRenameRejectsInvalidRequests(t *testing.T) {
	app, _ := testCLI(t)
	if err := app.run([]string{"add", "existing"}); err != nil {
		t.Fatal(err)
	}
	if err := app.run([]string{"add", "target"}); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"rename", "missing", "new-name"},
		{"rename", "existing", "target"},
		{"rename", "existing", "../unsafe"},
		{"rename", "existing", "existing"},
		{"rename", "existing"},
	} {
		if err := app.run(args); err == nil {
			t.Errorf("rename accepted invalid arguments %q", args)
		}
	}
	if _, err := os.Stat(filepath.Join(app.skillsDir(), "existing", "SKILL.md")); err != nil {
		t.Fatalf("invalid rename modified the source skill: %v", err)
	}
}

func TestSyncCreatesLinksAndResetRemovesOnlyRepositorySkills(t *testing.T) {
	app, _ := testCLI(t)
	source := filepath.Join(app.skillsDir(), "example")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("---\nname: example\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, destination := range app.destinations() {
		if err := os.MkdirAll(filepath.Join(destination, "unrelated"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if err := app.run([]string{"sync"}); err != nil {
		t.Fatal(err)
	}
	for _, destination := range app.destinations() {
		target := filepath.Join(destination, "example")
		info, err := os.Lstat(target)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("%s is not a symbolic link", target)
		}
		linked, err := os.Readlink(target)
		if err != nil {
			t.Fatal(err)
		}
		if linked != source {
			t.Fatalf("link points to %q, want %q", linked, source)
		}
	}

	// Reset must recognize repository links even after their source is deleted.
	if err := os.RemoveAll(source); err != nil {
		t.Fatal(err)
	}
	if err := app.run([]string{"reset"}); err != nil {
		t.Fatal(err)
	}
	for _, destination := range app.destinations() {
		if _, err := os.Lstat(filepath.Join(destination, "example")); !os.IsNotExist(err) {
			t.Fatalf("managed skill still exists: %v", err)
		}
		if _, err := os.Stat(filepath.Join(destination, "unrelated")); err != nil {
			t.Fatalf("reset removed unrelated skill: %v", err)
		}
	}
}

func TestSyncReplacesExistingDirectory(t *testing.T) {
	app, _ := testCLI(t)
	if err := os.MkdirAll(filepath.Join(app.skillsDir(), "example"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app.skillsDir(), "example", "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(app.destinations()[0], "example")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "stale"), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := app.sync(); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(target); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("existing directory was not replaced by a link: info=%v err=%v", info, err)
	}
}

func TestHelp(t *testing.T) {
	app, output := testCLI(t)
	if err := app.run([]string{"help"}); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"sync", "reset", "add <skill>", "delete <skill>", "rename <old> <new>", "help"} {
		if !strings.Contains(output.String(), command) {
			t.Errorf("help does not mention %q", command)
		}
	}
}
