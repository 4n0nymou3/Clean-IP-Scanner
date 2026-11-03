package utils

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/fatih/color"
)

const termuxSetupStorageCommand = "termux-setup-storage"

type ExportDeps struct {
	HomeDir      func() (string, error)
	LookPath     func(file string) (string, error)
	RunSetup     func(path string) error
	Rename       func(oldpath, newpath string) error
	PollInterval time.Duration
	PollTimeout  time.Duration
}

func DefaultExportDeps() ExportDeps {
	return ExportDeps{
		HomeDir:  os.UserHomeDir,
		LookPath: exec.LookPath,
		RunSetup: func(path string) error {
			cmd := exec.Command(path)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		},
		Rename:       os.Rename,
		PollInterval: time.Second,
		PollTimeout:  60 * time.Second,
	}
}

func downloadsDirWritable(dir string) bool {
	fi, err := os.Stat(dir)
	if err != nil || !fi.IsDir() {
		return false
	}
	probe, err := os.CreateTemp(dir, ".clean-ip-scanner-probe-*")
	if err != nil {
		return false
	}
	name := probe.Name()
	probe.Close()
	os.Remove(name)
	return true
}

func stageCopy(src, dstDir, name string) (string, error) {
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	fi, err := in.Stat()
	if err != nil {
		return "", err
	}
	if !fi.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file", src)
	}
	tmp, err := os.CreateTemp(dstDir, "."+name+".tmp-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	n, err := io.Copy(tmp, in)
	if err == nil && n != fi.Size() {
		err = fmt.Errorf("copied %d of %d bytes", n, fi.Size())
	}
	if err == nil {
		err = tmp.Sync()
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	os.Chmod(tmpPath, 0644)
	return tmpPath, nil
}

func waitForDownloads(dir string, deps ExportDeps) bool {
	deadline := time.Now().Add(deps.PollTimeout)
	for {
		if downloadsDirWritable(dir) {
			return true
		}
		if !time.Now().Before(deadline) {
			return false
		}
		time.Sleep(deps.PollInterval)
	}
}

func ExportResultsToDownloads(out io.Writer, deps ExportDeps, names []string) error {
	red := color.New(color.FgRed)
	yellow := color.New(color.FgYellow)
	green := color.New(color.FgGreen)
	white := color.New(color.FgWhite)
	cyan := color.New(color.FgCyan, color.Bold)

	fmt.Fprintln(out)

	home, err := deps.HomeDir()
	if err != nil || home == "" {
		yellow.Fprintln(out, "Could not export results to the Android Downloads folder: the home directory is unknown.")
		white.Fprintln(out, "Your result files are still saved in the scanner folder.")
		return fmt.Errorf("home directory unknown")
	}
	dir := filepath.Join(home, "storage", "downloads")

	if !downloadsDirWritable(dir) {
		setupPath, lookErr := deps.LookPath(termuxSetupStorageCommand)
		if lookErr != nil {
			yellow.Fprintln(out, "Could not export results to the Android Downloads folder.")
			white.Fprintln(out, "Shared storage is not accessible and the termux-setup-storage command was not found.")
			white.Fprintln(out, "Automatic export works inside Termux on Android.")
			white.Fprintln(out, "Your result files are still saved in the scanner folder.")
			return fmt.Errorf("storage unavailable: %v", lookErr)
		}
		cyan.Fprintln(out, "Android storage access is needed to copy your results to the Downloads folder.")
		white.Fprintln(out, "Requesting storage permission now (termux-setup-storage)...")
		yellow.Fprintln(out, "Android may show a permission request. Please tap Allow.")
		fmt.Fprintln(out)
		if runErr := deps.RunSetup(setupPath); runErr != nil {
			yellow.Fprintf(out, "termux-setup-storage reported a problem: %v\n", runErr)
		}
		white.Fprintln(out, "Waiting for storage access to be approved...")
		if !waitForDownloads(dir, deps) {
			fmt.Fprintln(out)
			red.Fprintln(out, "Storage access was not granted, so the results were not copied to Downloads.")
			white.Fprintln(out, "Your result files are still saved in the scanner folder.")
			yellow.Fprintln(out, "How to fix:")
			white.Fprintln(out, "  1. Run: termux-setup-storage  and tap Allow when Android asks.")
			white.Fprintln(out, "     If the request does not appear, open Android Settings > Apps > Termux")
			white.Fprintln(out, "     > Permissions and allow Files / Storage access.")
			white.Fprintln(out, "  2. Run the scanner again. Results will then be copied automatically.")
			return fmt.Errorf("storage access not granted")
		}
		green.Fprintln(out, "Storage access granted.")
	}

	type staged struct {
		name string
		tmp  string
		dst  string
	}
	var pending []staged
	cleanup := func() {
		for _, p := range pending {
			if p.tmp != "" {
				os.Remove(p.tmp)
			}
		}
	}

	for _, name := range names {
		tmp, err := stageCopy(name, dir, name)
		if err != nil {
			cleanup()
			red.Fprintf(out, "Could not copy %s to the Downloads folder: %v\n", name, err)
			white.Fprintln(out, "Nothing in the Downloads folder was changed. Your result files are still saved in the scanner folder.")
			return fmt.Errorf("copy failed for %s: %v", name, err)
		}
		pending = append(pending, staged{name: name, tmp: tmp, dst: filepath.Join(dir, name)})
	}

	var done []string
	for i := range pending {
		if err := deps.Rename(pending[i].tmp, pending[i].dst); err != nil {
			cleanup()
			red.Fprintf(out, "Could not finish exporting %s: %v\n", pending[i].name, err)
			if len(done) > 0 {
				yellow.Fprintln(out, "Already exported:")
				for _, d := range done {
					white.Fprintf(out, "  %s\n", d)
				}
			}
			white.Fprintln(out, "Your result files are still saved in the scanner folder.")
			return fmt.Errorf("export failed for %s: %v", pending[i].name, err)
		}
		pending[i].tmp = ""
		done = append(done, pending[i].dst)
	}

	green.Fprintln(out, "Results exported to Android Downloads:")
	for _, d := range done {
		green.Fprintf(out, "  %s\n", d)
	}
	return nil
}