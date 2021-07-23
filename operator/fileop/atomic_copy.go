package fileop

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/otiai10/copy"
	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/state"
)

type AtomicCopyOpts struct {
	Sources      []string `json:"sources"`
	Dest         string   `json:"dest"`
	ExcludeGlobs []string `json:"exclude,omitempty"`
}

type AtomicCopy struct {
	Args interface{}
}

func (op AtomicCopy) String() string {
	opts := op.Args.(*AtomicCopyOpts)

	return fmt.Sprintf("%s%s%s",
		strings.Join(opts.Sources, " "),
		padArg(true),
		opts.Dest,
	)
}

func (op AtomicCopy) Info() operator.Info {
	opts := op.Args.(*AtomicCopyOpts)

	cmd := &cobra.Command{
		Use:   "atomic-copy source... dest",
		Args:  cobra.MinimumNArgs(2),
		Short: "Copies files atomically",
		Long: `Atomic file copy. Sources can be globs.

Copies file(s) into an intermediate temporary location before executing a move,
which is atomic on linux systems.

This operator is similar to, but differs from cp -r in that, when multiple
sources are provided, and dest does not exist, it will be created.

For example, with cp -r, where directories a and b exist, but c does not:

$ cp -r a b c
cp: target 'c' is not a directory

atomic-copy differs in this case: It will create the directory with a and b
copied into it.

With a single source, cp -r will create the directory:

$ cp -r a c
$ tree -d
.
├── a
├── b
└── c

If the directory already exists, cp -r will copy a single source into it, but
atomic-copy will replace the dest directory with source.

atomic-copy is the same in this case.

If the directory exists, cp -r will copy all sources into dest:

$ cp -r a b c
$ tree -d
.
├── a
├── b
└── c
    ├── a
    └── b

atomic copy is the same in this case.
`,
	}

	flags := cmd.Flags()
	flags.StringArrayVar(&opts.ExcludeGlobs, "exclude", nil, "`glob`s to exclude from destination")

	return &operator.InfoData{
		OpName: "atomic-copy",
		Command: &operator.Command{
			Command:   cmd,
			ApplyArgs: atomicCopyArgs,
			Target:    opts,
		},
	}
}

func (op AtomicCopy) GetState(octx operator.Context) (state.State, error) {
	opts := op.Args.(*AtomicCopyOpts)
	st, err := getStateFileGlobs(octx.FS, state.State{}, opts.Dest, opts.Sources, opts.ExcludeGlobs)
	return st, err
}

func (op AtomicCopy) Run(octx operator.Context) error {
	opts := op.Args.(*AtomicCopyOpts)
	tmpDir, err := os.MkdirTemp("", "polyester-copy")
	if err != nil {
		return err
	}
	allFiles, err := gatherFilesGlobDirOnly(octx.FS, opts.Sources, opts.ExcludeGlobs)
	if err != nil {
		return err
	}

	if len(allFiles) == 0 {
		return fmt.Errorf("no files matched pattern(s): %v", opts.Sources)
	} else if len(allFiles) == 1 {
		file := allFiles[0]
		info, err := octx.FS.Stat(file)
		if err != nil {
			return err
		}
		src := octx.FS.Join(file)
		_, destFile := filepath.Split(file)
		dest := filepath.Join(tmpDir, destFile)
		// fmt.Println("copy", src, "->", dest)
		if info.IsDir() {
			if err := copy.Copy(src, dest); err != nil {
				return err
			}
		} else {
			if err := copyFile(src, dest); err != nil {
				return err
			}
		}
	} else {
		_, destFile := filepath.Split(opts.Dest)
		tmpDest := filepath.Join(tmpDir, destFile)
		for _, file := range allFiles {
			info, err := octx.FS.Stat(file)
			if err != nil {
				return err
			}
			src := octx.FS.Join(file)
			_, destFile := filepath.Split(file)
			dest := filepath.Join(tmpDest, destFile)

			if info.IsDir() {
				if err := copy.Copy(src, dest); err != nil {
					return err
				}
			} else {
				if err := copyFile(src, dest); err != nil {
					return err
				}
			}
		}
	}

	var srcPath string
	if len(allFiles) == 1 {
		srcPath = allFiles[0]
	} else {
		srcPath = opts.Dest
	}
	_, srcFile := filepath.Split(srcPath)
	src := filepath.Join(tmpDir, srcFile)

	destInfo, err := octx.FS.Stat(opts.Dest)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	destPath := octx.FS.Join(opts.Dest)
	var tmpDirPath string
	if destInfo != nil && destInfo.IsDir() {
		_, destFile := filepath.Split(opts.Dest)
		tmpDirPath = filepath.Join(tmpDir, destFile+".old")
		if err := os.Rename(destPath, tmpDirPath); err != nil {
			return err
		}
	}

	if err := os.Rename(src, destPath); err != nil {
		// try to move the intermediate dir back
		if tmpDirPath != "" {
			if rerr := os.Rename(tmpDirPath, destPath); rerr != nil {
				return fmt.Errorf("recovery failed: %v, orig error: %w", rerr, err)
			}
		}
		return fmt.Errorf("(recovered) %w", err)
	}

	return os.RemoveAll(tmpDir)
}

func excluded(p string, globs []string) (bool, error) {
	for _, glob := range globs {
		if ok, err := doublestar.Match(glob, p); err != nil {
			return ok, err
		} else if ok {
			return ok, nil
		}
	}
	return false, nil
}

func atomicCopyArgs(cmd *cobra.Command, args []string, target interface{}) error {
	t := target.(*AtomicCopyOpts)
	end := len(args) - 1
	t.Sources = args[:end]
	t.Dest = args[end]
	return nil
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
