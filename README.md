# execod

> Execute a command on demand

A simple listen loop that runs a command upon receiving a connection

Example use case: `execod go test ./mypackage` in one tmux split, Vim in
another. Type `<leader>x` to signal `execod` to run the test command.

Vim config:

```vim
nmap <leader>x :silent exec "!echo \| nc -U /tmp/execod.sock"<cr>
```

## Build

1. Install dependencies:

    ```bash
    $ dep ensure
    ```

2. Compile:

    ```bash
    $ go build
    ```

3. Run:

    ```bash
    $ ./execod uptime
    $ echo | nc -U /tmp/execod.sock
    ```
