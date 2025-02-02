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
```
Usage: threadinator [options]

Options:
  -c int        Number of concurrent threads
  -e string     Semicolon-separated commands to execute
  -p            Enable pipeline mode
  -t int        Timeout duration in seconds (default: defined in config.json)
  -v            Enable verbose output
  -cfg string   Change default settings (Must be in JSON syntax)
  -V            Show tool version
  -h, --help    Show help message
```

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

