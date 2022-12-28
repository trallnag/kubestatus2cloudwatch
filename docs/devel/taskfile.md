# Taskfile

Task is a task runner / build tool that aims to be simpler and easier to use
than, for example, GNU Make.

- <https://taskfile.dev/>
- <https://github.com/go-task/task>

Having Task installed is not a hard-requirement for developing Token2go. It is
mainly used to collect common scripts / commands. It is also used within GitHub
Actions.

It can be installed Homebrew (other options are available as well).

```
brew install go-task/tap/go-task
```

Task is configured via [`Taskfile.yaml`](../../Taskfile.yaml).

When adding new tasks to the task file, try to keep individual tasks simple and
small. More complicated things should be put into individual scripts and then
just called from Task.

## Cheat Sheet

List tasks.

```
task --list
```

Run task.

```
task update-swagger
```

Run task and set variable.

```
task update-swagger VERSION=4.15.5
```
