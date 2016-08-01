package lib

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

/*
unpack is an opinionated unpacker

It acts relative to the current working directory as follows:

1. creates a subdirectory with the name of the archive file - its extension.
2. moves the archive file to this folder.
3. extracts the content of the archive to this folder.
4. (optional) removes the archive file
5. flattens the folder hierarchy, i.e. if there is just one subfolder within the target folder, moves
   the content of the subfolder to the target folder and removes the subfolder.

The command also may act upon all files of known extensions of a directory or files that matches a regexp pattern.

It is just a wrapper around certain uncompressing commands that are executed in a subshell.

Here is a table of the supported file extensions and the expected commands.

-----------------------------
file ending | expected command inside the path
-----------------------------
tar         | tar
tgz         | tar, gzip
gz          | gzip
7z          | 7z
zip         | unzip
rar         | unrar

*/

var unpackerValidator = regexp.MustCompile(regexp.QuoteMeta("[FILE]"))

// registers the given cmd for the given extension. extension must start with '.'
// fileOpt is the commands parameter indicater for the file that is to be extracted
// pass fileOpt == "" for filename as last parameter
func RegisterUnpacker(ext string, cmd string) error {
	unpackerMX.Lock()
	defer unpackerMX.Unlock()

	if ext == "" {
		return fmt.Errorf("ext is empty")
	}

	if strings.IndexRune(ext, '.') != 0 {
		return fmt.Errorf("ext does not start with .")
	}

	if !unpackerValidator.MatchString(cmd) {
		return fmt.Errorf("cmd does not contain [FILE] placeholder")
	}

	if _, has := unpacker[strings.ToLower(ext)]; has {
		return UnpackerRegisteredError(strings.ToLower(ext))
	}

	unpacker[strings.ToLower(ext)] = cmd
	return nil
}

func HasUnpacker(ext string) (has bool) {
	_, has = unpacker[strings.ToLower(ext)]
	return
}

var infoLogger = log.New(os.Stdout, "unpack [INFO]", log.LstdFlags)
var verboseLogger = log.New(os.Stdout, "unpack [DEBUG]", log.LstdFlags)
var errorLogger = log.New(os.Stdout, "unpack [ERROR]", log.LstdFlags)

func logInfo(loglevel int, msg string) {
	if loglevel < 1 {
		return
	}
	infoLogger.Println(msg)
}

func logVerbose(loglevel int, msg string) {
	if loglevel < 2 {
		return
	}
	verboseLogger.Println(msg)
}

func logError(loglevel int, msg string) {
	if loglevel < 0 {
		return
	}
	errorLogger.Println(msg)
}

// remove removes file after successful extraction
// removeDirs are typical directories to be removed within extracted files, like __MACOSX, .git and .svn
// logleves: -1 = no logging
//            0 = error logging
//            1 = info logging
//            2 = verbose logging
// it will also try to "flatten" the directory, i.e. if there is just one single folder in it
// the content of this folder will be moved one folder up
func UnpackFile(filename string, dir string, remove bool, removeDirs []string, loglevel int) error {
	finfo, err := os.Stat(filepath.Join(dir, filename))

	if err != nil {
		logError(loglevel, err.Error())
		return err
	}

	if finfo.IsDir() {
		err = fmt.Errorf("is directory: %#v ", filename)
		logError(loglevel, err.Error())
		return err
	}

	ext := filepath.Ext(filename)

	if ext == "" {
		err = NoExtensionError(filepath.Join(dir, filename))
		logError(loglevel, err.Error())
		return err
	}

	p := unpacker[strings.ToLower(ext)]

	if len(p) == 0 {
		err = UnknownPackerError(strings.ToLower(ext))
		logError(loglevel, err.Error())
		return err
	}

	return UnpackFileWithUnpacker(filename, dir, p, remove, removeDirs, loglevel)
}

// unpacker slice contains the command itself at index 0 the option for the file at index 1
// and the other options in the order of the rest of the slice
// remove removes file after successful extraction
// unpacker is the string that is to be executed in a subshell. it must contain [FILE] as placeholder for
// the file that is to be extracted
// rmDirs are typical directories to be removed within extracted files, like __MACOSX, .git and .svn
// logleves: -1 = no logging
//            0 = error logging
//            1 = info logging
//            2 = verbose logging
// it will also try to "flatten" the directory, i.e. if there is just one single folder in it
// the content of this folder will be moved one folder up
func UnpackFileWithUnpacker(filename string, dir string, unpacker string, remove bool, rmDirs []string, loglevel int) error {
	createdDir, err := mkDir(filename, dir, loglevel)
	if err != nil {
		logError(loglevel, err.Error())
		return err
	}

	err = os.Rename(filepath.Join(dir, filename), filepath.Join(createdDir, filename))

	if err != nil {
		logError(loglevel, err.Error())
		return err
	}

	logVerbose(loglevel, fmt.Sprintf("moved %#v to %#v", filepath.Join(dir, filename), createdDir))

	err = runPackerCMD(createdDir, strings.Replace(unpacker, "[FILE]", filename, -1), loglevel)

	if err != nil {
		logError(loglevel, err.Error())
		return err
	}

	if remove {
		err = os.Remove(filepath.Join(createdDir, filename))
		if err != nil {
			logError(loglevel, err.Error())
			return err
		}
		logInfo(loglevel, fmt.Sprintf("removed %#v", filename))
	}

	if len(rmDirs) > 0 {
		removeDirs(createdDir, rmDirs, loglevel)
	}

	err = flatten(filename, createdDir, loglevel)
	if err != nil {
		logError(loglevel, err.Error())
		return err
	}

	return nil
}

// maps fileending to command and args
var unpacker = map[string]string{}

var unpackerMX = sync.Mutex{}

func mkDir(filename string, parentDir string, loglevel int) (createdDir string, err error) {
	ext := filepath.Ext(filename)
	if ext == "" {
		return "", NoExtensionError(filepath.Join(parentDir, filename))
	}

	r := regexp.MustCompile(regexp.QuoteMeta(ext) + "$")
	d := r.ReplaceAllString(filename, "")
	return mkDirTry(filepath.Join(parentDir, d), -1, loglevel)
}

func mkDirTry(dir string, try int, loglevel int) (createddir string, err error) {
	if try == 10 {
		return "", MkDirError(dir)
	}
	try += 1
	createddir = dir

	if try > 0 {
		createddir = fmt.Sprintf(dir+"-%d", try)
	}

	if os.Mkdir(createddir, 0755) != nil {
		logVerbose(loglevel, fmt.Sprintf("could not create dir %#v", createddir))
		return mkDirTry(dir, try, loglevel)
	}
	logInfo(loglevel, fmt.Sprintf("created dir %#v", createddir))
	return
}

// pass fileOpt == "" for filename as last parameter
func runPackerCMD(directory string, cmd string, loglevel int) error {
	//println(cmd + strings.Join(o, " "))
	c := exec.Command("/bin/sh", "-c", cmd)
	c.Dir = directory
	logInfo(loglevel, fmt.Sprintf("running command\n  %#v\n in directory\n  %#v\n ", cmd, directory))
	if loglevel > -1 {
		c.Stderr = os.Stderr
	}

	if loglevel > 1 {
		c.Stdout = os.Stdout
	}

	err := c.Run()
	if err != nil {
		return &RunError{
			Command: cmd,
			Err:     err,
		}
	}
	return nil
}

func removeDirs(dir string, subdirs []string, loglevel int) {
	for _, sub := range subdirs {
		path := filepath.Join(dir, sub)
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			logInfo(loglevel, fmt.Sprintf("removing %#v\n", path))
			os.RemoveAll(path)
		}
	}
}

func getDirContentsWithoutArchivFile(dir string, archivFile string) (res []os.FileInfo, err error) {
	var finfos []os.FileInfo

	finfos, err = ioutil.ReadDir(dir)

	if err != nil {
		return nil, err
	}

	for _, finfo := range finfos {
		if finfo.IsDir() || finfo.Name() != archivFile {
			res = append(res, finfo)
		}
	}

	return res, nil

}

func _flatten(archivfile string, dir string, sub string, loglevel int) error {
	d := fmt.Sprintf(dir+"-%d", time.Now().Nanosecond())

	logVerbose(loglevel, fmt.Sprintf("moving\n  %#v\nto\n  %#v\n", dir, d))
	err := os.Rename(dir, d)

	if err != nil {
		return err
	}

	logVerbose(loglevel, fmt.Sprintf("moving\n  %#v\nto\n  %#v\n", filepath.Join(d, sub), dir))
	err = os.Rename(filepath.Join(d, sub), dir)

	if err != nil {
		return err
	}

	finfo, err := os.Stat(filepath.Join(d, archivfile))

	if err == nil && !finfo.IsDir() {
		logVerbose(loglevel, fmt.Sprintf("moving\n  %#v\nto\n  %#v\n", filepath.Join(d, archivfile), filepath.Join(dir, archivfile)))
		err = os.Rename(filepath.Join(d, archivfile), filepath.Join(dir, archivfile))

		if err != nil {
			return err
		}
	}

	logVerbose(loglevel, fmt.Sprintf("removing\n  %#v\n", d))
	return os.Remove(d)
}

func flatten(archivFile string, dir string, loglevel int) (err error) {

	dir, err = filepath.Abs(dir)

	if err != nil {
		return err
	}

	var finfos []os.FileInfo

	finfos, err = getDirContentsWithoutArchivFile(dir, archivFile)

	if err != nil {
		return err
	}

	if len(finfos) == 1 && finfos[0].IsDir() {

		oldParent := finfos[0].Name()

		logInfo(loglevel, fmt.Sprintf("moving files from\n  %#v\nto \n %#v\n", filepath.Join(dir, oldParent), dir))
		return _flatten(archivFile, dir, oldParent, loglevel)
		/*
			err = os.Rename(filepath.Join(dir, oldParent), dir))

			finfos, err := ioutil.ReadDir(filepath.Join(dir, oldParent))

			if err != nil {
				return err
			}

			logInfo(loglevel, fmt.Sprintf("moving files from\n  %#v\nto \n %#v\n", filepath.Join(dir, oldParent), dir))

			for _, finfo := range finfos {
				logVerbose(loglevel, fmt.Sprintf("moving\n  %#v\nto\n  %#v\n", filepath.Join(dir, oldParent, finfo.Name()), filepath.Join(dir, finfo.Name())))
				err = os.Rename(filepath.Join(dir, oldParent, finfo.Name()), filepath.Join(dir, finfo.Name()))
				if err != nil {
					return err
				}
			}

			logInfo(loglevel, fmt.Sprintf("removing %#v\n", filepath.Join(dir, oldParent)))
			return os.Remove(filepath.Join(dir, oldParent))
		*/
	}
	return nil
}
