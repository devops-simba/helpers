package helpers

import (
	"os"
)

type Path string

func (this Path) Stat() (os.FileInfo, error) {
	return os.Stat(string(this))
}
func (this Path) Exists() bool {
	_, err := this.Stat()
	if err != nil {
		return !os.IsNotExist(err)
	}
	return true
}
func (this Path) IsDir() bool {
	stat, err := this.Stat()
	if err != nil {
		return false
	}
	return stat.IsDir()
}
func (this Path) IsFile() bool {
	stat, err := this.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & (os.ModeDir | os.ModeDevice | os.ModeNamedPipe | os.ModeSocket)) == 0
}
func (this Path) IsSymlink() bool {
	stat, err := this.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeSymlink) != 0
}

func PathExists(path string) bool    { return Path(path).Exists() }
func PathIsDir(path string) bool     { return Path(path).IsDir() }
func PathIsFile(path string) bool    { return Path(path).IsFile() }
func PathIsSymlink(path string) bool { return Path(path).IsSymlink() }
