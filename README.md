find_dupes
==========

command-line utility to identify duplicate files in a directory tree

installation
------------

    go get github.com/zackse/find_dupes

description
-----------

This program crawls a directory tree and finds duplicate files. First, files
with identical sizes are grouped together, then within each group, files with
duplicate md5sums are emitted to standard output.

usage
-----

```bash
find_dupes <PATH> [ <NUM_WORKERS> ]
```

`NUM_WORKERS` defaults to 2. Note that this limit only affects the number of
goroutines collecting file sizes (calling `os.Stat()`) in parallel. For each
file size with more than one match, the program will launch a goroutine for
every duplicate entry to generate its md5sum.

example
-------

```bash
find_dupes ~/Pictures 4
```
