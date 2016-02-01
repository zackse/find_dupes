find_dupes
==========

command-line utility to identify duplicate files in a directory tree

installation
------------

    go get github.com/zackse/find_dupes/find_dupes

description
-----------

This program crawls a directory tree and finds duplicate files. The first
pass groups files with identical sizes, then the second pass groups files
with duplicate MD5sums.

usage
-----

```bash
find_dupes <PATH> [ <NUM_WORKERS> ]
```

`NUM_WORKERS` defaults to 2.

example
-------

```bash
find_dupes ~/Pictures 4
```
