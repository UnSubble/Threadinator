# Threadinator

## Overview
Threadinator is a command-line tool designed for executing multiple commands concurrently. It provides features such as pipeline execution, configuration through JSON files, and various command-line options.

## Features
- Execute commands concurrently using multiple threads.
- Support for pipeline execution.
- JSON-based configuration management.
- Customizable timeout durations.
- Verbose mode for detailed output.
- Command-line options for dynamic configuration.

## Installation
Clone the repository and build the project using Go:

```bash
$ git clone https://github.com/unsubble/threadinator.git
$ cd threadinator
$ go build -o threadinator
```

## Usage
### Command Line Options
- `-e, --execute`: Semicolon-separated commands to execute.
- `-c, --count`: Number of concurrent threads.
- `-p, --pipeline`: Enable pipeline mode.
- `-v, --verbose`: Enable verbose output.
- `--log-level`: Set the logging level (INFO, DEBUG, WARN, ERROR).
- `-t, --timeout`: Timeout duration in seconds.
- `--cfg`: Change default settings (must be in JSON syntax).
- `-V, --version`: Show tool version.

### Example Commands
Run multiple commands concurrently:

```bash
$ ./threadinator -e "echo Hello; echo World" -c 2 -v
```

Change configuration settings dynamically:

```bash
$ ./threadinator -cfg '{"timeout": 15, "timeunit": "ms"}'
```

Display version information:

```bash
$ ./threadinator -V
```

Run commands with a custom timeout:

```bash
$ ./threadinator -e "sleep 5; echo Done" -t 4
```

Enable pipeline mode:

```bash
$ ./threadinator -e "echo Hello; grep H" -p
```

Set the logging level to DEBUG:

```bash
$ ./threadinator -e "echo Debugging" --log-level DEBUG
```

Work with dependency of commands:
```bash
$ threadinator -p -e "xargs echo:1|0|1; echo Test"
```

Work with commands with random delay:
```bash
$ threadinator -p -e "echo Random time:rand(1,5)|1"
```

### Configuration
The tool uses a `config.json` file to store default settings. The configuration file has the following format:

```json
{
  "name": "threadinator",
  "timeout": 10,
  "timeunit": "s",
  "version": "1.0.0",
  "thread-count": 5,
  "verbose": false,
  "pipeline": false
}
```

## Contributing
Contributions are welcome! Please follow these steps:
1. Fork the repository.
2. Create a new branch for your feature or bug fix.
3. Commit your changes with clear and descriptive messages.
4. Push to your fork and submit a pull request.

## License
This project is licensed under the [MIT License](LICENSE).

## Acknowledgments
Special thanks to all contributors who made this project possible.

