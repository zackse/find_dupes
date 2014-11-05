find_dupes
==========

command-line utility to identify duplicate files in a directory tree

installation
------------

    go get github.com/zackse/find_dupes/...

usage
-----

```bash
find_dupes_merge_maps <PATH> [ <NUM_WORKERS> ]
find_dupes_shared_map <PATH> [ <NUM_WORKERS> ]
```

`NUM_WORKERS` defaults to 2.

example
-------

```bash
find_dupes_merge_maps ~/Pictures 4
```

