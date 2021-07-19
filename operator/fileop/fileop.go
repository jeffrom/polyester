// Package fileop contains filesystem-related operators.
//
// TODO have cp op that works exactly the same as gnu cp, then have cp-dir,
// cp-atomic, cp-plan, cp-secret
package fileop

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/otiai10/copy"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/operator/opfs"
	"github.com/jeffrom/polyester/state"
)

func Checksum(p string) ([]byte, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ChecksumReader(f)
}

func ChecksumReader(r io.Reader) ([]byte, error) {
	sha := sha256.New()
	if _, err := io.Copy(sha, r); err != nil {
		return nil, err
	}

	return sha.Sum(nil), nil
}

func getStateFileGlobs(ofs operator.FS, st state.State, dest string, globs, excludes []string) (state.State, error) {
	allFiles, err := gatherFilesGlob(ofs, globs, excludes)
	if err != nil {
		return st, err
	}

	st, err = appendFiles(ofs, st, true, false, allFiles...)
	if err != nil {
		return st, err
	}

	info, err := ofs.Stat(dest)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return st, err
	}

	dests := []string{dest}
	if info != nil && info.IsDir() {
		var err error
		dests, err = gatherFilesDir(ofs, []string{dest}, excludes)
		if err != nil {
			return st, err
		}
	}

	st, err = appendFiles(ofs, st, true, true, dests...)
	if err != nil {
		return st, err
	}
	return st, nil
}

// appendFiles appends files to the state, include full mode and checksum.
func appendFiles(ofs operator.FS, st state.State, source, target bool, files ...string) (state.State, error) {
	for _, file := range files {
		info, err := ofs.Stat(file)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return st, err
		}

		var checksum []byte
		if info != nil && !info.IsDir() {
			var err error
			checksum, err = Checksum(ofs.Join(file))
			if err != nil {
				return st, err
			}
		}

		if source {
			st = st.Append(state.Entry{
				Name: file,
				File: &opfs.StateFileEntry{
					Info:   info,
					SHA256: checksum,
				},
			})
		}
		if target {
			st = st.Append(state.Entry{
				Name:   file,
				Target: true,
				File: &opfs.StateFileEntry{
					Info:   info,
					SHA256: checksum,
				},
			})
		}
	}
	return st, nil
}

func gatherFilesGlobDirOnly(ofs operator.FS, globs, excludes []string) ([]string, error) {
	var allFiles []string
	for _, srcpat := range globs {
		files, err := ofs.Glob(srcpat)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			if excl, err := excluded(file, excludes); err != nil {
				return nil, err
			} else if excl {
				continue
			}
			allFiles = append(allFiles, file)
		}
	}
	return allFiles, nil
}

func gatherFilesGlob(ofs operator.FS, globs, excludes []string) ([]string, error) {
	var allFiles []string

	for _, pat := range globs {
		files, err := ofs.Glob(pat)
		if err != nil {
			return nil, err
		}
		next, err := gatherFiles(ofs, files, excludes)
		if err != nil {
			return nil, err
		}
		allFiles = append(allFiles, next...)
	}
	return allFiles, nil
}

func gatherFiles(ofs operator.FS, files, excludes []string) ([]string, error) {
	var allFiles []string
	for _, file := range files {
		info, err := ofs.Stat(file)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if info == nil || !info.IsDir() {
			allFiles = append(allFiles, file)
			continue
		}

		dirFiles, err := gatherFilesDir(ofs, []string{file}, excludes)
		if err != nil {
			return nil, err
		}
		allFiles = append(allFiles, dirFiles...)
	}
	return allFiles, nil
}

func gatherFilesDir(ofs operator.FS, files, excludes []string) ([]string, error) {
	var allFiles []string
	walkFn := func(p string, d fs.DirEntry, perr error) error {
		if perr != nil {
			return perr
		}
		if excl, err := excluded(p, excludes); err != nil {
			return err
		} else if excl {
			return nil
		}
		allFiles = append(allFiles, p)
		return nil
	}
	for _, file := range files {
		if err := fs.WalkDir(ofs, file, walkFn); err != nil {
			return nil, fmt.Errorf("walkdir failed: %w", err)
		}
	}
	return allFiles, nil
}

func copyOneOrManyFiles(ofs operator.FS, destFile string, sources []string) error {
	if len(sources) == 0 {
		return errors.New("no files found")
	}
	if len(sources) == 1 {
		return copyOneFile(ofs, sources[0], destFile)
	}
	return copyManyFiles(ofs, sources, destFile)
}

func copyOneFile(ofs operator.FS, file, destFile string) error {
	// fmt.Println("copyOneFile", file, destFile)
	srcInfo, err := ofs.Stat(file)
	if err != nil {
		return err
	}
	destInfo, err := ofs.Stat(destFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if destInfo != nil && srcInfo.IsDir() != destInfo.IsDir() {
		return fmt.Errorf("source (%s) and dest (%s) are different types", file, destFile)
	}

	src := file
	dest := destFile
	if srcInfo.IsDir() {
		if err := copy.Copy(src, dest); err != nil {
			return err
		}
	} else {
		if err := copyFile(src, dest); err != nil {
			return err
		}
	}
	return nil
}

func copyManyFiles(ofs operator.FS, sources []string, destDir string) error {
	dest := destDir
	for _, file := range sources {
		info, err := ofs.Stat(file)
		if err != nil {
			return err
		}
		_, srcFile := filepath.Split(file)
		src := file

		destPath := filepath.Join(dest, srcFile)
		if info.IsDir() {
			if err := copy.Copy(src, destPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(src, destPath); err != nil {
				return err
			}
		}
	}

	return nil
}
