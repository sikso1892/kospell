# This script runs the kospell-cli Go application.

# You can pass arguments directly to the Go application.
# For example, to check a file:
# ./run.ps1 -f text.txt

# You can also pipe text to this script.
# For example:
# "안녕 하세요" | ./run.ps1

# Or you can pass the text as an argument
# ./run.ps1 "안녕 하세요"
# ./run.ps1 안녕 하세요

if ($MyInvocation.ExpectingInput) {
    # If there is input from the pipeline, pipe it to the Go application.
    $input | go run ./cmd/kospell-cli
} else {
    # If the first argument does not start with a hyphen, treat all arguments as a single string to be checked.
    if ($args.Count -gt 0 -and $args[0] -notlike "-*") {
        ($args -join " ") | go run ./cmd/kospell-cli
    } else {
        # Otherwise, pass the arguments to the Go application (e.g., for file flags).
        go run ./cmd/kospell-cli $args
    }
}
