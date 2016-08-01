unpack is an opinionated unpacker

# what is it

I was annoyed when downloading compressed files from the internets that when extracting them, some of them have everything packed nicely within an dedicated subfolder while others not.

So to have my hard-disk clean, I have to look into every archive before unpacking it in order so see, if I must create a dedicated folder for it.

So this is a simple tool that does the job, i.e. makes it so that I always have a dedicated folder for a decompressed archive.

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

```
-----------------------------
file ending | expected command inside the path
-----------------------------
tar         | tar
tgz         | tar, gzip
gz          | gzip
7z          | 7z
zip         | unzip
rar         | unrar
```


# install 

WARNING: Use it at your own risk!! It might delete your files!!!!

To install

`go get https://github.com/metakeule/unpack`

# usage

`unpack help`

# other

The underlying library also can be used like the following 

```go

package main

import "github.com/metakeule/unpack/unpack.v1"

func main() {
    unpacker := unpack.New()
    err := unpacker.UnpackFile("myfile.zip")
    ....
}
```

For documentation, see: https://godoc.org/github.com/metakeule/unpack/unpack.v1