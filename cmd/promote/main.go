package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/homeadmin/internal/config"
	"github.com/homeadmin/internal/database"
	"github.com/homeadmin/internal/repositories"
)

func promoteUser(repo repositories.UserRepository, email string) error {
	user, err := repo.FindByEmail(email)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("user not found with email %q", email)
	}
	user.IsAdmin = true
	return repo.Update(user)
}

func main() {
	emailFlag := flag.String("email", "", "Email of the user to promote to site admin")
	flag.Parse()

	if *emailFlag == "" {
		fmt.Fprintln(os.Stderr, "Error: email is required")
		flag.Usage()
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	dbConn, err := database.ConnectWithDriver(cfg.DatabaseURL, cfg.DBDriver)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to database: %v\n", err)
		os.Exit(1)
	}

	userRepo := repositories.NewUserRepository(dbConn)
	if err := promoteUser(userRepo, *emailFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully promoted user %q to site admin\n", *emailFlag)
}
