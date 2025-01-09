# go-filewatcher

A basic filewatcher utility that executes a couple of commands when it notices file changes in given directories.

## How to use

`filewatcher --init-cmd="echo Start" --prep-cmd="echo Updated" --run-cmd="echo Running" --extensions=".foo,.baz" /path/to/dir/to/watch1 dir/to/watch2`

- The `--init-cmd` is executed when the utility runs, leave empty to skip.
- The `--prep-cmd` is executed when the utility notice a filesystem change, and is awaited completion. Also optional.
- The `--run-cmd` is executed after the `--prep-cmd` completes. This one is mandatory.
- With the `--extensions` flag you can control which file extensions to trigger on (comma separated).

The non-positional arguments to the utility determine the directories to watch for changes (recursively).

If no arguments are given, it defaults to the working directory.

## Why?

Why write this when there are already so many of these out there you ask? Well, when using colima on MacOS, I ran into this issue:

https://github.com/abiosoft/colima/issues/1244
