package util

import (
    "io"
    "os"
    "path/filepath"
)

/*
Exists reports whether the named file or directory exists.
*/
func Exists(name string) bool {
    if _, err := os.Stat(name); err != nil {
        if os.IsNotExist(err) {
            return false
        }
    }
    return true
}

/*
Mv copies the file at src to the new file at dst.
Credit: https://gist.github.com/elazarl/5507969
*/
func Mv(dst, src string) error {
    s, err := os.Open(src)
    if err != nil {
        return err
    }
    // no need to check errors on read only file, we already got everything
    // we need from the filesystem, so nothing can go wrong now.
    defer s.Close()
    d, err := os.Create(dst)
    if err != nil {
        return err
    }
    if _, err := io.Copy(d, s); err != nil {
        d.Close()
        return err
    }
    if err := d.Close(); err != nil {
        return err
    }

    if err:= os.Remove(src); err != nil {
        return err
    }

    return nil
}

/*
MoveTo moves the file at fname to the directory dir.
*/
func MoveTo(dir, fname string) error {
    dst := filepath.Join(dir, fname)
    return Mv(dst, fname)
}


