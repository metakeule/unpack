package main

import (
	"bytes"
	"fmt"
	"github.com/metakeule/config"
	"github.com/metakeule/unpack/unpack.v1"
	"io"
	"os"
	"path/filepath"
)

var (
	cfg = config.MustNew(
		"unpack",
		"1.0.0",
		`unpack is an opinionated unpacker

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

`,
	)

	fileArg = cfg.NewString(
		"file",
		"archive file to be extracted",
		config.Shortflag('f'),
	)

	verbosityArg = cfg.NewInt32(
		"verbose",
		"verbosity level of logging: -1 = no logging, 0 = error logging, 1 = info logging, 2 = verbose logging",
		config.Shortflag('v'),
		config.Default(int32(0)),
	)

	rmArg = cfg.NewBool(
		"rm",
		"remove the archive file after successful extraction",
		config.Shortflag('r'),
		config.Default(false),
	)

	// __MACOSX
	rmMACOSXArg = cfg.NewBool(
		"rmmacosx",
		"remove __MACOSX directories",
		config.Default(true),
	)

	rmGitArg = cfg.NewBool(
		"rmgit",
		"remove .git directories",
		config.Default(false),
	)

	rmSvnArg = cfg.NewBool(
		"rmsvn",
		"remove .svn directories",
		config.Default(false),
	)

	dirArg = cfg.NewBool(
		"dir",
		"extract all files in the working directory",
		config.Shortflag('d'),
	)

	matchArg = cfg.NewString(
		"match",
		"extract all files in the working directory that are matching the pattern (regular expression)",
		config.Shortflag('m'),
	)
)

func main() {
	reportError(run())
}

func run() (err error) {
	var (
		wd       string
		options  []unpack.Option
		unpacker interface {
			UnpackFile(string) error
			UnpackAllFiles(string) map[string]error
			UnpackFilesMatching(dir string, pattern string) map[string]error
		}
	)

steps:
	for jump := 1; err == nil; jump++ {
		switch jump - 1 {
		default:
			break steps
		// count a number up for each following step
		case 0:
			wd, err = os.Getwd()
		case 1:
			wd, err = filepath.Abs(wd)
		case 2:
			err = cfg.Run()
		case 3:
			switch verbosityArg.Get() {
			case -1:
				// do nothing, i.e. no logging
			case 1:
				options = append(options, unpack.LogInfos)
			case 2:
				options = append(options, unpack.LogVerbose)
			default:
				// error logging, also == 0
				options = append(options, unpack.LogErrors)
			}
		case 4:
			if rmdirs := getRmDirs(); len(rmdirs) > 0 {
				options = append(options, unpack.RemoveDirectories(rmdirs...))
			}
		case 5:
			if rmArg.Get() {
				options = append(options, unpack.RemoveArchive)
			}
		case 6:
			unpacker = unpack.New(options...)
		case 7:
			if matchArg.IsSet() {
				errs := unpacker.UnpackFilesMatching(wd, matchArg.Get())
				if len(errs) > 0 {
					err = &errorMap{errs}
				}
				break steps
			}
		case 8:
			if dirArg.Get() {
				errs := unpacker.UnpackAllFiles(wd)
				if len(errs) > 0 {
					err = &errorMap{errs}
				}
				break steps
			}
		case 9:
			if !fileArg.IsSet() {
				err = fmt.Errorf("missing file argument")
			}
		case 10:
			err = unpacker.UnpackFile(fileArg.Get())
		}
	}

	return
}

func getRmDirs() (rmdirs []string) {
	if rmMACOSXArg.Get() {
		rmdirs = append(rmdirs, "__MACOSX")
	}
	if rmGitArg.Get() {
		rmdirs = append(rmdirs, ".git")
	}
	if rmSvnArg.Get() {
		rmdirs = append(rmdirs, ".svn")
	}
	return
}

func reportError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR!")
		if m, ok := err.(*errorMap); ok {
			m.WriteTo(os.Stderr)
			return
		}

		fmt.Fprintln(os.Stderr, err.Error())
	}
}

type errorMap struct {
	errs map[string]error
}

func (e *errorMap) WriteTo(w io.Writer) {
	for k, v := range e.errs {
		fmt.Fprintf(w, "## %s ##\n%s\n\n", k, v.Error())
	}
}

func (e *errorMap) Error() string {
	var bf bytes.Buffer
	e.WriteTo(&bf)
	return bf.String()
}
