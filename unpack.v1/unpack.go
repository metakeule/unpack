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

// RegisterUnpacker registers the given cmd for the given extension ext.
// ext must start with "." like e.g. ".zip"
// cmd must contain [FILE] placeholder for filename, e.g. "unzip [FILE]"
func RegisterUnpacker(ext string, cmd string) error {
	return lib.RegisterUnpacker(ext, cmd)
}

// MustRegisterUnpacker is like RegisterUnpacker but panicks if there is an error.
func MustRegisterUnpacker(ext string, cmd string) {
	err := RegisterUnpacker(ext, cmd)
	if err != nil {
		panic(err.Error())
	}
}

// RemoveArchive is an Option that removes the archive file after successful unpacking.
// It is meant to be passed to New().
var RemoveArchive Option = func(c *config) {
	c.removeArchive = true
}

// RemoveDirectories returns an Option that removes typical directories to be removed within extracted files, like __MACOSX, .git and .svn.
// It is meant to be passed to New().
func RemoveDirectories(dirs ...string) Option {
	return func(c *config) {
		c.rmDirs = dirs
	}
}

// LogVerbose is an Option that enables verbose logging. This also includes error logging and info logging.
// It is meant to be passed to New().
var LogVerbose Option = func(c *config) {
	c.logLevel = 2
}

// LogErrors is an Option that enables error logging.
// It is meant to be passed to New().
var LogErrors Option = func(c *config) {
	c.logLevel = 0
}

// LogInfos is an Option that enables info logging. This also includes error logging.
// It is meant to be passed to New().
var LogInfos Option = func(c *config) {
	c.logLevel = 1
}

// Option is a configuration option that is meant to be passed to New().
type Option func(*config)

// New returns a new unpacker.
// By default, logging is disabled. To enable it, pass one of the logging options as parameter.
// New accepts options of type Option to enabled configuration.
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

// UnpackFile unpacks the given file into a subdirectory which is named after the file (- its extension)
// The subdirectory is created in the same folder where file resides.
// Before unpacking, the file is moved to the subdirectory.
// After uncompressing, the content of the subdirectory will be flattened by one level, i.e.
// If there is just one subdirectory, its content is moved one level up.
// it will also try to "flatten" the directory, i.e. if there is just one single folder in it
// the content of this folder will be moved one folder up.
// If RemoveArchive was set, file is removed after successful unpacking.
// Any directories set via RemoveDirectories will be removed inside the unpacked directory.
func (c *config) UnpackFile(file string) (err error) {
	file, err = filepath.Abs(file)
	if err != nil {
		return
	}
	return lib.UnpackFile(filepath.Base(file), filepath.Dir(file), c.removeArchive, c.rmDirs, c.logLevel)
}

// UnpackAllFiles is like UnpackFile, but acting on all files with an extension for which a unpacker command
// has been registered. By default that includes: ".tgz",".tar",".zip",".rar",".7z",".gz"
// Make sure the corresponding command is available since otherwise in the middle of the processing there will
// be a problem when the command is executed. If so that function returns at a state when the archive file has
// been moved to the newly created folder (see documentation of UnpackFile).
func (c *config) UnpackAllFiles(dir string) (errors map[string]error) {
	return c.unpackFilesInDir(dir, fileHasUnpacker)
}

// UnpackFilesMatching is like UnpackAllFiles but only affects the files that are matching the given pattern.
// The pattern must be a valid regular expression.
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
