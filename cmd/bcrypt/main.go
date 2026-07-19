package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const defaultBcryptCost = 12

func main() {
	plain, err := readPlaintext(os.Args[1:])
	if err != nil {
		fatal(err)
	}

	cost, err := resolveCost()
	if err != nil {
		fatal(err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		fatal(fmt.Errorf("generate bcrypt hash: %w", err))
	}

	fmt.Println(string(hash))
}

func readPlaintext(args []string) (string, error) {
	if len(args) > 0 {
		value := strings.Join(args, " ")
		if strings.TrimSpace(value) == "" {
			return "", errors.New("plaintext must not be empty")
		}
		return value, nil
	}

	info, err := os.Stdin.Stat()
	if err != nil {
		return "", fmt.Errorf("stat stdin: %w", err)
	}

	if (info.Mode() & os.ModeCharDevice) != 0 {
		return "", errors.New("usage: go run ./cmd/bcrypt -- <plaintext>  or  printf '<plaintext>' | go run ./cmd/bcrypt")
	}

	reader := bufio.NewReader(os.Stdin)
	value, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, os.ErrClosed) {
		value = strings.TrimSpace(value)
		if value != "" {
			return value, nil
		}
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("plaintext must not be empty")
	}

	return value, nil
}

func resolveCost() (int, error) {
	raw := strings.TrimSpace(os.Getenv("BCRYPT_COST"))
	if raw == "" {
		return defaultBcryptCost, nil
	}

	cost, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid BCRYPT_COST %q: %w", raw, err)
	}

	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return 0, fmt.Errorf("BCRYPT_COST must be between %d and %d", bcrypt.MinCost, bcrypt.MaxCost)
	}

	return cost, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
