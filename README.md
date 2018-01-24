# execod

> Execute a command on demand

A simple listen loop that runs a command upon receiving a connection

Example use case: `execod go test ./mypackage` in one tmux split, Vim in
another. Type `<leader>x` to signal `execod` to run the test command.

Vim config:

```vim
nmap <leader>x :silent exec "!echo \| nc -U /tmp/execod.sock"<cr>
```
