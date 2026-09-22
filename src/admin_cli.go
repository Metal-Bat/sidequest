package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Metal-Bat/sidequest/internal/account"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/term"
)

func createAdmin(ctx context.Context, pool *pgxpool.Pool) error {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return errors.New("create-admin requires an interactive terminal")
	}
	fmt.Print("Admin username: ")
	username, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return err
	}
	fmt.Print("Password (hidden): ")
	password, err := term.ReadPassword(fd)
	fmt.Println()
	if err != nil {
		return err
	}
	defer clear(password)
	fmt.Print("Confirm password: ")
	confirm, err := term.ReadPassword(fd)
	fmt.Println()
	if err != nil {
		return err
	}
	defer clear(confirm)
	if string(password) != string(confirm) {
		return errors.New("passwords do not match")
	}
	err = account.Create(ctx, pool, strings.TrimSpace(username), string(password), true)
	if errors.Is(err, account.ErrExists) || errors.Is(err, account.ErrInvalid) {
		return err
	}
	if err != nil {
		return errors.New("could not create admin; check database availability and run mise run migrate first")
	}
	fmt.Println("Admin created. Sign in and open Create user to add regular accounts.")
	return nil
}
