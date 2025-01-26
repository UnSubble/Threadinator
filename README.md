# Threadinator

**Threadinator** is a command-line tool designed to run CLI commands in parallel using multiple threads. It supports features like pipelining, verbose logging, error handling, and much more.

## Features

- **Parallel Execution**: Run multiple instances of a command in parallel with configurable thread count.
- **Pipelining**: Pipe the output of one command to the next.
- **Verbose Logging**: Enable detailed logs for each thread execution.
- **Error Handling**: Get descriptive error messages and manage null pointer checks.
- **Timeout Support**: Configure a timeout for each thread.
- **Output Redirection**: Redirect the output of commands to a log file.
- **Dynamic Thread Count**: Automatically adjust the number of threads based on available CPU cores.
- **Resource Limiting**: Set CPU and memory usage limits for commands.
- **Pretty Logs**: Enable colorful and readable logs for better debugging.
- **Statistics**: View a summary of the execution including successes, errors, and timing.

## Installation

### Building from Source

1. Clone the repository:

   ```bash
   git clone https://github.com/yourusername/threadinator.git
   cd threadinator
   ```

2. Build the project:
   ```bash
   go build -o threadinator .
   ```


## Usage

### Basic Command Syntax
   ```bash
   threadinator -c THREAD_COUNT -e "COMMAND" [OPTIONS]
```

- `-c THREAD_COUNT`: Number of threads (workers) to run concurrently.
- `-e COMMAND`: The command to execute.
- `-a ARGS`: Arguments to pass to the command.
- `-p`: Enable pipelining between threads.
- `-v`: Enable verbose logging for detailed output.
- `-o OUTPUT_FILE`: Redirect the output to a file.
- `-t TIMEOUT`: Set a timeout (in seconds) for each thread.

### Examples

#### Basic Usage

Run a command on 4 threads:
```bash
threadinator -c 4 -e "echo hello world"
```
## Contributing

If you'd like to contribute to **Threadinator**, feel free to fork the repository, create a branch, and submit a pull request. Contributions are always welcome!

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.