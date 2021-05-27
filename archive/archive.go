package archive

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/nwaples/rardecode"
)

type Filter func(name string) bool

type Archive interface {
	Extract(dest string, filter Filter) (map[string]string, error)
}

type ZipArchive struct {
	zip *zip.Reader
}
type RarArchive struct {
	rar *rardecode.Reader
}

func NewReader(mime string, r io.Reader, size int64) (Archive, error) {
	var err error
	var a Archive

	switch mime {
	case "application/x-rar-compressed":
		r, err := rardecode.NewReader(r, "")
		if err != nil {
			return a, err
		}

		return &RarArchive{r}, nil
	case "application/x-zip-compressed":
		r := io.LimitReader(r, size)
		buf := make([]byte, size)
		_, err = io.ReadFull(r, buf)
		if err != nil {
			return a, err
		}

		z, err := zip.NewReader(bytes.NewReader(buf), size)
		if err != nil {
			return a, err
		}

		return &ZipArchive{z}, nil
	}

	return a, fmt.Errorf("mimetype '%s' not supported", mime)
}

type dest struct {
	d   string
	dir bool
}

func newDest(path string) *dest {
	stat, _ := os.Stat(path)
	isDir := stat != nil && stat.IsDir()
	return &dest{path, isDir}
}

func (d *dest) file(name string) (ok bool, tmp, real string) {
	real = d.d
	if d.dir {
		real = filepath.Join(d.d, name)
	}

	stat, _ := os.Stat(real)
	ok = stat == nil

	tmp = real + ".tmp"
	return
}

func (d *dest) single() bool { return !d.dir }

func (d *dest) read(r io.Reader, name string) (bool, string, error) {
	ok, tmp, real := d.file(name)
	if !ok {
		return false, real, nil
	}

	f, err := os.Create(tmp)
	if err != nil {
		return false, real, err
	}
	if _, err = io.Copy(f, r); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return false, real, err
	}

	_ = f.Close()
	if err = os.Rename(tmp, real); err != nil {
		_ = os.Remove(tmp)
		return false, real, err
	}

	return true, real, nil
}

func (r *RarArchive) Extract(dest string, filter Filter) (map[string]string, error) {
	files := make(map[string]string)

	destination := newDest(dest)

	for {
		inode, err := r.rar.Next()
		if err != nil {
			if err == io.EOF {
				err = nil
			}

			return files, err
		}
		if inode.IsDir {
			continue
		}

		clean := filepath.Base(filepath.Clean(inode.Name))
		files[clean] = ""

		if !filter(clean) {
			continue
		}

		ok, fn, err := destination.read(r.rar, clean)
		if err != nil {
			return files, err
		}

		if ok {
			files[clean] = fn
			if destination.single() {
				break
			}
		}
	}

	return files, nil
}

func (z *ZipArchive) Extract(dest string, filter Filter) (map[string]string, error) {
	files := make(map[string]string, len(z.zip.File))
	for _, inode := range z.zip.File {
		files[inode.Name] = ""
	}

	destination := newDest(dest)

	for _, inode := range z.zip.File {
		clean := filepath.Base(filepath.Clean(inode.Name))
		if !filter(clean) {
			continue
		}

		zf, err := z.zip.Open(inode.Name)
		if err != nil {
			return files, err
		}

		ok, fn, err := destination.read(zf, clean)
		if err != nil {
			return files, err
		}

		if ok {
			files[clean] = fn
			if destination.single() {
				break
			}
		}
	}

	return files, nil
}
