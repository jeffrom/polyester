package manifest

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var headerBytes = []byte("+vRE4eUD3Mi53e6J4sE6wKE42UBR5EJrnjeffROm=")
var utf8bom = []byte{0xEF, 0xBB, 0xBF}
var drivePathPattern = regexp.MustCompile(`^[a-zA-Z]:/`)

func LoadFile(name string) (*Manifest, error) {
	if info, err := os.Stat(name); err != nil {
		return nil, err
	} else if info.IsDir() {
		return nil, errors.New("cannot load a directory")
	}

	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if err := checkTar(name, f); err != nil {
		return nil, err
	}

	m, err := LoadArchive(f)
	if err != nil {
		if err == gzip.ErrHeader {
			return nil, fmt.Errorf("file '%s' does not appear to be a valid chart file (err: %s)", name, err)
		}
	}
	return m, err
}

func LoadArchive(r io.Reader) (*Manifest, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	tfs := newTarFS("")
	tr := tar.NewReader(zr)
	for {
		b := bytes.NewBuffer(nil)
		hd, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if hd.FileInfo().IsDir() {
			// Use this instead of hd.Typeflag because we don't have to do any
			// inference chasing.
			continue
		}

		switch hd.Typeflag {
		// We don't want to process these extension header files.
		case tar.TypeXGlobalHeader, tar.TypeXHeader:
			continue
		}

		// Archive could contain \ if generated on Windows
		delimiter := "/"
		if strings.ContainsRune(hd.Name, '\\') {
			delimiter = "\\"
		}

		parts := strings.Split(hd.Name, delimiter)
		// n := strings.Join(parts[1:], delimiter)
		n := strings.Join(parts, delimiter)

		// Normalize the path to the / delimiter
		n = strings.ReplaceAll(n, delimiter, "/")

		if path.IsAbs(n) {
			return nil, errors.New("manifest illegally contains absolute paths")
		}

		n = path.Clean(n)
		if n == "." {
			// In this case, the original path was relative when it should have been absolute.
			return nil, fmt.Errorf("manifest illegally contains content outside the base directory: %q", hd.Name)
		}
		if strings.HasPrefix(n, "..") {
			return nil, errors.New("manifest illegally references parent directory")
		}

		// In some particularly arcane acts of path creativity, it is possible to intermix
		// UNIX and Windows style paths in such a way that you produce a result of the form
		// c:/foo even after all the built-in absolute path checks. So we explicitly check
		// for this condition.
		if drivePathPattern.MatchString(n) {
			return nil, errors.New("manifest contains illegally named files")
		}

		// limit to 256mb, that should be fine right??
		if _, err := io.Copy(b, io.LimitReader(tr, 1024*1024*256)); err != nil {
			return nil, err
		}

		data := bytes.TrimPrefix(b.Bytes(), utf8bom)

		// fmt.Println("read file:", n, "length:", len(data))
		tfs.AddFile(&tarFile{
			name:     n,
			contents: data,
			mode:     0644,
			modTime:  time.Now(),
		})
	}
	return loadFS(tfs, "polyester.sh")
}

func Save(m *Manifest, destDir string) (string, error) {
	if err := m.Validate(); err != nil {
		return "", fmt.Errorf("manifest invalid: %w", err)
	}
	filename := "manifest.tgz"
	if meta := m.Metadata; meta != nil {
		filename = fmt.Sprintf("%s-%s.tgz", meta.Name, meta.Version)
	}
	filename = filepath.Join(destDir, filename)
	dir := filepath.Dir(filename)
	if info, err := os.Stat(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if merr := os.MkdirAll(dir, 0755); merr != nil {
				return "", merr
			}
		} else {
			return "", err
		}
	} else if !info.IsDir() {
		return "", fmt.Errorf("manifest: not a directory: %s", destDir)
	}

	f, err := os.Create(filename)
	if err != nil {
		return "", err
	}

	zw := gzip.NewWriter(f)
	zw.Header.Extra = headerBytes
	zw.Header.Comment = "Polyester"

	tw := tar.NewWriter(zw)
	rollback := false
	defer func() {
		tw.Close()
		zw.Close()
		f.Close()
		if rollback {
			os.Remove(filename)
		}
	}()

	if err := writeTar(tw, m, ""); err != nil {
		rollback = true
		return filename, err
	}
	return filename, nil
}

func checkTar(name string, f *os.File) error {
	defer f.Seek(0, 0)

	// Check the file format to give us a chance to provide the user with more actionable feedback.
	buffer := make([]byte, 512)
	_, err := f.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("file '%s' cannot be read: %s", name, err)
	}
	if contentType := http.DetectContentType(buffer); contentType != "application/x-gzip" {
		// TODO: Is there a way to reliably test if a file content is YAML? ghodss/yaml accepts a wide
		//       variety of content (Makefile, .zshrc) as valid YAML without errors.

		// Wrong content type. Let's check if it's yaml and give an extra hint?
		if strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml") {
			return fmt.Errorf("file '%s' seems to be a YAML file, but expected a gzipped archive", name)
		}
		return fmt.Errorf("file '%s' does not appear to be a gzipped archive; got '%s'", name, contentType)
	}
	return nil
}

func writeTar(out *tar.Writer, m *Manifest, prefix string) error {
	base := prefix
	if err := writeToTar(out, filepath.Join(base, m.Main), m.MainScript); err != nil {
		return err
	}

	if err := writeFilesToTar(out, filepath.Join(base, "files"), m.Files); err != nil {
		return err
	}
	if err := writeFilesToTar(out, filepath.Join(base, "templates"), m.Templates); err != nil {
		return err
	}
	if err := writeFilesToTar(out, filepath.Join(base, "vars"), m.Vars); err != nil {
		return err
	}
	if err := writeFilesToTar(out, filepath.Join(base, "secrets"), m.Secrets); err != nil {
		return err
	}

	keys := make([]string, len(m.Plans))
	i := 0
	for k := range m.Plans {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, k := range keys {
		plan := m.Plans[k]
		if err := writeTar(out, plan, filepath.Join("plans", k)); err != nil {
			return err
		}
	}
	return nil
}

func writeFilesToTar(out *tar.Writer, base string, m map[string][]byte) error {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, k := range keys {
		b := m[k]
		if err := writeToTar(out, filepath.Join(base, k), b); err != nil {
			return err
		}
	}
	return nil
}

func writeToTar(out *tar.Writer, name string, body []byte) error {
	h := &tar.Header{
		Name:    filepath.ToSlash(name),
		Mode:    0644,
		Size:    int64(len(body)),
		ModTime: time.Now(),
	}
	if err := out.WriteHeader(h); err != nil {
		return err
	}
	_, err := out.Write(body)
	return err
}
