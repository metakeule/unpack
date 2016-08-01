package unpack

import (
	"io/ioutil"
	"lib"
	"path/filepath"
	"regexp"
)

func init() {
	MustRegisterUnpacker(".tgz", "tar -xzf [FILE]")
	MustRegisterUnpacker(".tar", "tar -xf [FILE]")
	MustRegisterUnpacker(".zip", "unzip [FILE]")
	MustRegisterUnpacker(".rar", "unrar x [FILE]")
	MustRegisterUnpacker(".7z", "7z x [FILE]")
	MustRegisterUnpacker(".gz", "gzip -d [FILE]")

}

// registers the given cmd for the given extension. extension must start with '.'
// cmd must contain [FILE] placeholder for filename
func RegisterUnpacker(ext string, cmd string) error {
	return lib.RegisterUnpacker(ext, cmd)
}

// cmd must contain [FILE] placeholder for filename
func MustRegisterUnpacker(ext string, cmd string) {
	err := RegisterUnpacker(ext, cmd)
	if err != nil {
		panic(err.Error())
	}
}

// RemoveArchive removes the archive file after successful extraction
var RemoveArchive Option = func(c *config) {
	c.removeArchive = true
}

// RemoveDirectories removes typical directories to be removed within extracted files, like __MACOSX, .git and .svn
func RemoveDirectories(dirs ...string) Option {
	return func(c *config) {
		c.rmDirs = dirs
	}
}

var LogVerbose Option = func(c *config) {
	c.logLevel = 2
}

var LogErrors Option = func(c *config) {
	c.logLevel = 0
}

var LogInfos Option = func(c *config) {
	c.logLevel = 1
}

type Option func(*config)

// by default, no logging is enabled
func New(opts ...Option) interface {
	UnpackFile(string) error
	UnpackAllFiles(string) map[string]error
	UnpackFilesMatching(dir string, pattern string) map[string]error
} {
	c := &config{}
	c.logLevel = -1

	for _, opt := range opts {
		opt(c)
	}

	return c
}

type config struct {
	removeArchive bool
	rmDirs        []string
	logLevel      int
}

// file can be either relative to cwd of absolute path
// the files are unpacked in to the newly created folder with the name of the archive file - extension
// the archive file is also moved into this folder
// it will also try to "flatten" the directory, i.e. if there is just one single folder in it
// the content of this folder will be moved one folder up
func (c *config) UnpackFile(file string) (err error) {
	file, err = filepath.Abs(file)
	if err != nil {
		return
	}
	return lib.UnpackFile(filepath.Base(file), filepath.Dir(file), c.removeArchive, c.rmDirs, c.logLevel)
}

// unpacks all files that have a known unpacker
// dir can be either relative or absolute path
// the files are unpacked in to the newly created folder with the name of the archive file - extension
// the archive file is also moved into this folder
// it will try to "flatten" the directory, i.e. if there is just one single folder in it
// the content of this folder will be moved one folder up
func (c *config) UnpackAllFiles(dir string) (errors map[string]error) {
	return c.unpackFilesInDir(dir, fileHasUnpacker)
}

// unpacks all files which filename matches the pattern
// dir can be either relative or absolute path
// the files are unpacked in to the newly created folder with the name of the archive file - extension
// the archive file is also moved into this folder
// it will try to "flatten" the directory, i.e. if there is just one single folder in it
// the content of this folder will be moved one folder up
func (c *config) UnpackFilesMatching(dir string, pattern string) (errors map[string]error) {
	r, err := regexp.Compile(pattern)

	if err != nil {
		return map[string]error{
			pattern: err,
		}
	}

	cb := func(fname string) bool {
		return r.MatchString(fname)
	}

	return c.unpackFilesInDir(dir, cb)
}

func fileHasUnpacker(file string) bool {
	return lib.HasUnpacker(filepath.Ext(file))
}

// callback is a function that gets a filename and returns true if the file should be unpacked
func (c *config) unpackFilesInDir(dir string, callback func(fname string) bool) (errors map[string]error) {
	errs := map[string]error{}

	finfos, err := ioutil.ReadDir(dir)

	if err != nil {
		errs[dir] = err
		return errs
	}

	for _, finfo := range finfos {
		if !finfo.IsDir() && callback(finfo.Name()) {
			fErr := c.UnpackFile(filepath.Join(dir, finfo.Name()))

			if fErr != nil {
				errs[filepath.Join(dir, finfo.Name())] = fErr
			}
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}
