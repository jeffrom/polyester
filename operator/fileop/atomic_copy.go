package fileop

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/otiai10/copy"
	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/operator/opfs"
)

type AtomicCopyOpts struct {
	Sources      []string `json:"sources"`
	Dest         string   `json:"dest"`
	ExcludeGlobs []string `json:"exclude,omitempty"`
}

type AtomicCopy struct {
	Args interface{}
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

func (op AtomicCopy) GetState(octx operator.Context) (operator.State, error) {
	opts := op.Args.(*AtomicCopyOpts)
	st := operator.State{}
	var allFiles []string
	for _, srcpat := range opts.Sources {
		files, err := octx.FS.Glob(srcpat)
		if err != nil {
			return st, err
		}
		for _, file := range files {
			if excl, err := excluded(file, opts.ExcludeGlobs); err != nil {
				return st, err
			} else if excl {
				continue
			}
			allFiles = append(allFiles, file)
			// TODO probably need to traverse dirs
		}
	}

	for _, file := range allFiles {
		// fmt.Println("GetState file:", file, octx.FS.Join(file))
		info, err := octx.FS.Stat(file)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return st, err
		}

		var checksum []byte
		if info != nil && !info.IsDir() {
			var err error
			checksum, err = Checksum(file)
			if err != nil {
				return st, err
			}
		}

		st = st.Append(operator.StateEntry{
			Name: file,
			File: &opfs.StateFileEntry{
				Info:   info,
				SHA256: checksum,
			},
		})

	}

	info, err := octx.FS.Stat(opts.Dest)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return st, err
	}
	var checksum []byte
	if info != nil && !info.IsDir() {
		var err error
		checksum, err = Checksum(opts.Dest)
		if err != nil {
			return st, err
		}
	}

	// include both source and target state here, since we want to rerun if the
	// target changes.
	st = st.Append(operator.StateEntry{
		Name: opts.Dest,
		File: &opfs.StateFileEntry{
			Info:   info,
			SHA256: checksum,
		},
	})
	st = st.Append(operator.StateEntry{
		Name:   opts.Dest,
		Target: true,
		File: &opfs.StateFileEntry{
			Info:   info,
			SHA256: checksum,
		},
	})

	return st, nil
}

func (op AtomicCopy) Run(octx operator.Context) error {
	opts := op.Args.(*AtomicCopyOpts)
	tmpDir, err := os.MkdirTemp("", "polyester-copy")
	if err != nil {
		return err
	}
	var allFiles []string
	for _, srcpat := range opts.Sources {
		files, err := octx.FS.Glob(srcpat)
		if err != nil {
			return err
		}
		for _, file := range files {
			if excl, err := excluded(file, opts.ExcludeGlobs); err != nil {
				return err
			} else if excl {
				continue
			}
			allFiles = append(allFiles, file)
		}
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
				return fmt.Errorf("recovery failed: %w, orig error: %w", rerr, err)
			}
		}
		return fmt.Errorf("(recovered) %w", err)
	}

	return os.RemoveAll(tmpDir)
}

func atomicCopyArgs(cmd *cobra.Command, args []string, target interface{}) error {
	t := target.(*AtomicCopyOpts)
	end := len(args) - 1
	t.Sources = args[:end]
	t.Dest = args[end]
	return nil
}

func excluded(p string, globs []string) (bool, error) {
	return false, nil
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
