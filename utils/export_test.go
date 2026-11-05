package utils

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type exportEnv struct {
	home      string
	downloads string
	work      string
	setupRuns int
}

func newExportEnv(t *testing.T) *exportEnv {
	t.Helper()
	env := &exportEnv{home: t.TempDir(), work: t.TempDir()}
	env.downloads = filepath.Join(env.home, "storage", "downloads")
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(env.work); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
	return env
}

func (e *exportEnv) deps() ExportDeps {
	return ExportDeps{
		HomeDir:      func() (string, error) { return e.home, nil },
		LookPath:     func(string) (string, error) { return "/fake/termux-setup-storage", nil },
		RunSetup:     func(string) error { e.setupRuns++; return nil },
		Rename:       os.Rename,
		PollInterval: 5 * time.Millisecond,
		PollTimeout:  60 * time.Millisecond,
	}
}

func (e *exportEnv) grantStorage(t *testing.T) {
	t.Helper()
	if err := os.MkdirAll(e.downloads, 0755); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func tempLeftovers(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var found []string
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") || strings.Contains(e.Name(), "probe") {
			found = append(found, e.Name())
		}
	}
	return found
}

var resultNames = []string{"clean_ips.txt", "clean_ips_list.txt"}

func TestExportWhenStorageAlreadyWorks(t *testing.T) {
	env := newExportEnv(t)
	env.grantStorage(t)
	writeFile(t, "clean_ips.txt", "full results\nline 2\n")
	writeFile(t, "clean_ips_list.txt", "1.1.1.1\n--------------------\n1.1.1.1\n")
	var out bytes.Buffer
	if err := ExportResultsToDownloads(&out, env.deps(), resultNames); err != nil {
		t.Fatalf("export failed: %v\n%s", err, out.String())
	}
	if env.setupRuns != 0 {
		t.Fatal("termux-setup-storage must not run when storage already works")
	}
	for _, n := range resultNames {
		if readFile(t, filepath.Join(env.downloads, n)) != readFile(t, n) {
			t.Fatalf("%s was not copied exactly", n)
		}
	}
	for _, n := range resultNames {
		if !strings.Contains(out.String(), filepath.Join(env.downloads, n)) {
			t.Fatalf("success message must show %s destination:\n%s", n, out.String())
		}
	}
	if left := tempLeftovers(t, env.downloads); len(left) != 0 {
		t.Fatalf("leftover temp files: %v", left)
	}
	if readFile(t, "clean_ips.txt") != "full results\nline 2\n" {
		t.Fatal("local result file must be unchanged")
	}
}

func TestExportReplacesPreviousExport(t *testing.T) {
	env := newExportEnv(t)
	env.grantStorage(t)
	writeFile(t, filepath.Join(env.downloads, "clean_ips.txt"), "OLD OLD OLD OLD OLD OLD OLD OLD\n")
	writeFile(t, filepath.Join(env.downloads, "clean_ips_list.txt"), "OLD LIST\n")
	writeFile(t, "clean_ips.txt", "new\n")
	writeFile(t, "clean_ips_list.txt", "new list\n")
	if err := ExportResultsToDownloads(&bytes.Buffer{}, env.deps(), resultNames); err != nil {
		t.Fatal(err)
	}
	if readFile(t, filepath.Join(env.downloads, "clean_ips.txt")) != "new\n" {
		t.Fatal("old export must be replaced, not appended to")
	}
	if readFile(t, filepath.Join(env.downloads, "clean_ips_list.txt")) != "new list\n" {
		t.Fatal("old list export must be replaced")
	}
}

func TestExportRequestsStoragePermissionOnce(t *testing.T) {
	env := newExportEnv(t)
	writeFile(t, "clean_ips.txt", "a\n")
	writeFile(t, "clean_ips_list.txt", "b\n")
	deps := env.deps()
	deps.RunSetup = func(string) error {
		env.setupRuns++
		return os.MkdirAll(env.downloads, 0755)
	}
	var out bytes.Buffer
	if err := ExportResultsToDownloads(&out, deps, resultNames); err != nil {
		t.Fatalf("export failed: %v\n%s", err, out.String())
	}
	if env.setupRuns != 1 {
		t.Fatalf("expected exactly one permission request, got %d", env.setupRuns)
	}
	if !strings.Contains(out.String(), "tap Allow") {
		t.Fatalf("user must be told to tap Allow:\n%s", out.String())
	}
	if readFile(t, filepath.Join(env.downloads, "clean_ips.txt")) != "a\n" {
		t.Fatal("file not exported after permission grant")
	}
	env.setupRuns = 0
	if err := ExportResultsToDownloads(&bytes.Buffer{}, deps, resultNames); err != nil {
		t.Fatal(err)
	}
	if env.setupRuns != 0 {
		t.Fatal("second export must not request permission again")
	}
}

func TestExportWaitsForDelayedApproval(t *testing.T) {
	env := newExportEnv(t)
	writeFile(t, "clean_ips.txt", "a\n")
	writeFile(t, "clean_ips_list.txt", "b\n")
	deps := env.deps()
	deps.PollTimeout = 2 * time.Second
	deps.RunSetup = func(string) error {
		go func() {
			time.Sleep(80 * time.Millisecond)
			os.MkdirAll(env.downloads, 0755)
		}()
		return nil
	}
	if err := ExportResultsToDownloads(&bytes.Buffer{}, deps, resultNames); err != nil {
		t.Fatalf("delayed approval should still export: %v", err)
	}
}

func TestExportPermissionDeniedDoesNotCrashAndKeepsLocalFiles(t *testing.T) {
	env := newExportEnv(t)
	writeFile(t, "clean_ips.txt", "keep me\n")
	writeFile(t, "clean_ips_list.txt", "keep me too\n")
	var out bytes.Buffer
	err := ExportResultsToDownloads(&out, env.deps(), resultNames)
	if err == nil {
		t.Fatal("expected an error when permission is never granted")
	}
	if env.setupRuns != 1 {
		t.Fatalf("expected one request, got %d", env.setupRuns)
	}
	msg := out.String()
	if !strings.Contains(msg, "Storage access was not granted") || !strings.Contains(msg, "termux-setup-storage") {
		t.Fatalf("missing recovery guidance:\n%s", msg)
	}
	if readFile(t, "clean_ips.txt") != "keep me\n" || readFile(t, "clean_ips_list.txt") != "keep me too\n" {
		t.Fatal("local files must stay intact")
	}
	if _, statErr := os.Stat(env.downloads); statErr == nil {
		t.Fatal("nothing should be created in Downloads")
	}
}

func TestExportSetupCommandFailureIsReported(t *testing.T) {
	env := newExportEnv(t)
	writeFile(t, "clean_ips.txt", "a\n")
	writeFile(t, "clean_ips_list.txt", "b\n")
	deps := env.deps()
	deps.RunSetup = func(string) error { return errors.New("exit status 1") }
	var out bytes.Buffer
	if err := ExportResultsToDownloads(&out, deps, resultNames); err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(out.String(), "exit status 1") {
		t.Fatalf("setup failure must be mentioned:\n%s", out.String())
	}
}

func TestExportExplainsExistingStorageDirectory(t *testing.T) {
	env := newExportEnv(t)
	writeFile(t, "clean_ips.txt", "a\n")
	writeFile(t, "clean_ips_list.txt", "b\n")
	if err := os.MkdirAll(filepath.Join(env.home, "storage"), 0755); err != nil {
		t.Fatal(err)
	}
	if !storageDirExists(env.home) {
		t.Fatal("existing storage directory must be detected")
	}
	deps := env.deps()
	deps.RunSetup = func(string) error {
		env.setupRuns++
		return os.MkdirAll(env.downloads, 0755)
	}
	var out bytes.Buffer
	if err := ExportResultsToDownloads(&out, deps, resultNames); err != nil {
		t.Fatalf("export failed: %v\n%s", err, out.String())
	}
	msg := out.String()
	for _, want := range []string{
		"Your Termux storage directory already exists.",
		"Do you want to continue? (y/n)",
		"Type y and press Enter",
		"does not delete",
		"Step 1",
		"Step 2",
		"tap Allow",
		"does not grant this permission",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing %q in output:\n%s", want, msg)
		}
	}
}

func TestExportNoStorageDirectoryMessageWhenMissing(t *testing.T) {
	env := newExportEnv(t)
	writeFile(t, "clean_ips.txt", "a\n")
	writeFile(t, "clean_ips_list.txt", "b\n")
	if storageDirExists(env.home) {
		t.Fatal("storage directory must not exist in this test")
	}
	deps := env.deps()
	deps.RunSetup = func(string) error {
		env.setupRuns++
		return os.MkdirAll(env.downloads, 0755)
	}
	var out bytes.Buffer
	if err := ExportResultsToDownloads(&out, deps, resultNames); err != nil {
		t.Fatalf("export failed: %v\n%s", err, out.String())
	}
	msg := out.String()
	if strings.Contains(msg, "already exists") || strings.Contains(msg, "(y/n)") || strings.Contains(msg, "Step 1") {
		t.Fatalf("existing-directory explanation must not be shown:\n%s", msg)
	}
	if !strings.Contains(msg, "tap Allow") {
		t.Fatalf("Allow instruction missing:\n%s", msg)
	}
}

func TestExportNoSetupMessagesWhenStorageAlreadyWorks(t *testing.T) {
	env := newExportEnv(t)
	env.grantStorage(t)
	writeFile(t, "clean_ips.txt", "a\n")
	writeFile(t, "clean_ips_list.txt", "b\n")
	var out bytes.Buffer
	if err := ExportResultsToDownloads(&out, env.deps(), resultNames); err != nil {
		t.Fatal(err)
	}
	if env.setupRuns != 0 {
		t.Fatal("setup must not run")
	}
	msg := out.String()
	if strings.Contains(msg, "(y/n)") || strings.Contains(msg, "Allow") || strings.Contains(msg, "termux-setup-storage") {
		t.Fatalf("no setup messages expected:\n%s", msg)
	}
}

func TestSetupCommandKeepsTerminalConnected(t *testing.T) {
	cmd := newSetupCommand("/fake/termux-setup-storage")
	if cmd.Stdin != os.Stdin || cmd.Stdout != os.Stdout || cmd.Stderr != os.Stderr {
		t.Fatal("setup command must use the real terminal streams")
	}
	if cmd.SysProcAttr != nil && (cmd.SysProcAttr.Setsid || cmd.SysProcAttr.Setpgid) {
		t.Fatal("setup command must not be detached from the terminal")
	}
}

func TestExportCancelledSetupDoesNotWaitForTimeout(t *testing.T) {
	env := newExportEnv(t)
	writeFile(t, "clean_ips.txt", "keep a\n")
	writeFile(t, "clean_ips_list.txt", "keep b\n")
	deps := env.deps()
	deps.PollInterval = 50 * time.Millisecond
	deps.PollTimeout = 10 * time.Second
	deps.RunSetup = func(string) error { env.setupRuns++; return errors.New("exit status 1") }
	var out bytes.Buffer
	start := time.Now()
	err := ExportResultsToDownloads(&out, deps, resultNames)
	if err == nil {
		t.Fatal("expected error when setup is cancelled")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("cancelled setup must not wait for the poll timeout, took %v", elapsed)
	}
	msg := out.String()
	if !strings.Contains(msg, "Automatic export was cancelled") || !strings.Contains(msg, "Run the scanner again") {
		t.Fatalf("missing cancellation guidance:\n%s", msg)
	}
	if strings.Contains(msg, "Waiting for storage access") || strings.Contains(msg, "Results exported") {
		t.Fatalf("unexpected message:\n%s", msg)
	}
	if readFile(t, "clean_ips.txt") != "keep a\n" || readFile(t, "clean_ips_list.txt") != "keep b\n" {
		t.Fatal("local result files must stay intact")
	}
	if _, statErr := os.Stat(env.downloads); statErr == nil {
		t.Fatal("nothing must be created in Downloads")
	}
}

func TestExportNonZeroSetupExitStillExportsWhenWritable(t *testing.T) {
	env := newExportEnv(t)
	writeFile(t, "clean_ips.txt", "a\n")
	writeFile(t, "clean_ips_list.txt", "b\n")
	deps := env.deps()
	deps.RunSetup = func(string) error {
		env.setupRuns++
		if err := os.MkdirAll(env.downloads, 0755); err != nil {
			return err
		}
		return errors.New("exit status 1")
	}
	var out bytes.Buffer
	if err := ExportResultsToDownloads(&out, deps, resultNames); err != nil {
		t.Fatalf("export must continue when Downloads is writable: %v\n%s", err, out.String())
	}
	for _, n := range resultNames {
		if readFile(t, filepath.Join(env.downloads, n)) != readFile(t, n) {
			t.Fatalf("%s was not exported", n)
		}
	}
	if strings.Contains(out.String(), "cancelled") {
		t.Fatalf("must not report cancellation:\n%s", out.String())
	}
}

func TestExportWithoutTermuxCommand(t *testing.T) {
	env := newExportEnv(t)
	writeFile(t, "clean_ips.txt", "a\n")
	writeFile(t, "clean_ips_list.txt", "b\n")
	deps := env.deps()
	deps.LookPath = func(string) (string, error) { return "", errors.New("not found") }
	var out bytes.Buffer
	if err := ExportResultsToDownloads(&out, deps, resultNames); err == nil {
		t.Fatal("expected error outside Termux")
	}
	if env.setupRuns != 0 {
		t.Fatal("setup must not run when the command is missing")
	}
	if !strings.Contains(out.String(), "not found") {
		t.Fatalf("missing explanation:\n%s", out.String())
	}
}

func TestExportDownloadsPathNotADirectory(t *testing.T) {
	env := newExportEnv(t)
	writeFile(t, "clean_ips.txt", "a\n")
	writeFile(t, "clean_ips_list.txt", "b\n")
	if err := os.MkdirAll(filepath.Dir(env.downloads), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, env.downloads, "i am a file")
	if err := ExportResultsToDownloads(&bytes.Buffer{}, env.deps(), resultNames); err == nil {
		t.Fatal("expected error when Downloads is not a directory")
	}
	if readFile(t, env.downloads) != "i am a file" {
		t.Fatal("unrelated file must not be touched")
	}
}

func TestExportUnknownHome(t *testing.T) {
	env := newExportEnv(t)
	deps := env.deps()
	deps.HomeDir = func() (string, error) { return "", errors.New("no home") }
	if err := ExportResultsToDownloads(&bytes.Buffer{}, deps, resultNames); err == nil {
		t.Fatal("expected error")
	}
}

func TestExportMissingSourceChangesNothing(t *testing.T) {
	env := newExportEnv(t)
	env.grantStorage(t)
	writeFile(t, filepath.Join(env.downloads, "clean_ips.txt"), "previous export\n")
	writeFile(t, "clean_ips.txt", "fresh\n")
	var out bytes.Buffer
	err := ExportResultsToDownloads(&out, env.deps(), resultNames)
	if err == nil {
		t.Fatal("expected error when a source file is missing")
	}
	if readFile(t, filepath.Join(env.downloads, "clean_ips.txt")) != "previous export\n" {
		t.Fatal("existing export must stay intact when the pair cannot be exported")
	}
	if _, statErr := os.Stat(filepath.Join(env.downloads, "clean_ips_list.txt")); statErr == nil {
		t.Fatal("no partial export expected")
	}
	if left := tempLeftovers(t, env.downloads); len(left) != 0 {
		t.Fatalf("leftover temp files: %v", left)
	}
	if !strings.Contains(out.String(), "clean_ips_list.txt") {
		t.Fatalf("message must name the failing file:\n%s", out.String())
	}
}

func TestExportRenameFailureKeepsExistingFilesAndCleansUp(t *testing.T) {
	env := newExportEnv(t)
	env.grantStorage(t)
	writeFile(t, filepath.Join(env.downloads, "clean_ips.txt"), "previous\n")
	writeFile(t, filepath.Join(env.downloads, "clean_ips_list.txt"), "previous list\n")
	writeFile(t, "clean_ips.txt", "fresh\n")
	writeFile(t, "clean_ips_list.txt", "fresh list\n")
	deps := env.deps()
	deps.Rename = func(string, string) error { return errors.New("simulated failure") }
	var out bytes.Buffer
	if err := ExportResultsToDownloads(&out, deps, resultNames); err == nil {
		t.Fatal("expected error")
	}
	if readFile(t, filepath.Join(env.downloads, "clean_ips.txt")) != "previous\n" ||
		readFile(t, filepath.Join(env.downloads, "clean_ips_list.txt")) != "previous list\n" {
		t.Fatal("destination files must be untouched after a failed copy")
	}
	if left := tempLeftovers(t, env.downloads); len(left) != 0 {
		t.Fatalf("leftover temp files: %v", left)
	}
	if strings.Contains(out.String(), "Results exported") {
		t.Fatalf("must not claim success:\n%s", out.String())
	}
	if readFile(t, "clean_ips.txt") != "fresh\n" {
		t.Fatal("local files must stay intact")
	}
}

func TestExportSecondRenameFailureIsReportedAccurately(t *testing.T) {
	env := newExportEnv(t)
	env.grantStorage(t)
	writeFile(t, "clean_ips.txt", "fresh\n")
	writeFile(t, "clean_ips_list.txt", "fresh list\n")
	deps := env.deps()
	calls := 0
	deps.Rename = func(a, b string) error {
		calls++
		if calls == 2 {
			return errors.New("simulated failure")
		}
		return os.Rename(a, b)
	}
	var out bytes.Buffer
	if err := ExportResultsToDownloads(&out, deps, resultNames); err == nil {
		t.Fatal("expected error")
	}
	msg := out.String()
	if strings.Contains(msg, "Results exported") {
		t.Fatalf("must not claim full success:\n%s", msg)
	}
	if !strings.Contains(msg, "Already exported") || !strings.Contains(msg, "clean_ips_list.txt") {
		t.Fatalf("partial result must be described:\n%s", msg)
	}
	if left := tempLeftovers(t, env.downloads); len(left) != 0 {
		t.Fatalf("leftover temp files: %v", left)
	}
}

func TestDefaultExportDepsAreComplete(t *testing.T) {
	d := DefaultExportDeps()
	if d.HomeDir == nil || d.LookPath == nil || d.RunSetup == nil || d.Rename == nil {
		t.Fatal("default dependencies must all be set")
	}
	if d.PollInterval <= 0 || d.PollTimeout <= 0 {
		t.Fatal("polling durations must be positive")
	}
}