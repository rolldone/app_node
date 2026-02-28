package auth

import (
	"fmt"
	"log"
	"strconv"

	"golang.org/x/crypto/bcrypt"

	"go_framework/internal/db"
	authmodels "go_framework/plugins/auth/models"
	authservices "go_framework/plugins/auth/services"

	"github.com/spf13/cobra"
)

func (p *Plugin) ConsoleCommands() []*cobra.Command {
	newAdminService := func() *authservices.AdminService {
		gdb, err := db.GetGormDB()
		if err != nil || gdb == nil {
			log.Fatalf("db unavailable: %v", err)
		}
		svc, serr := authservices.NewAdminService(gdb)
		if serr != nil {
			log.Fatalf("service init: %v", serr)
		}
		return svc
	}

	adminCmd := &cobra.Command{
		Use:   "auth:admin",
		Short: "Admin CRUD commands for auth plugin",
	}

	var username, email, password, level string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create an admin",
		Run: func(cmd *cobra.Command, args []string) {
			if password == "" {
				fmt.Print("Password: ")
				fmt.Scanln(&password)
			}
			svc := newAdminService()
			hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				log.Fatalf("failed to hash password: %v", err)
			}
			admin := &authmodels.Admin{
				Username:     username,
				Email:        email,
				PasswordHash: string(hashed),
				Level:        level,
				IsActive:     true,
			}
			if err := svc.CreateAdmin(admin); err != nil {
				log.Fatalf("failed to create admin: %v", err)
			}
			fmt.Printf("created admin id=%s\n", admin.ID)
		},
	}
	createCmd.Flags().StringVar(&username, "username", "", "admin username (required)")
	createCmd.Flags().StringVar(&email, "email", "", "admin email (required)")
	createCmd.Flags().StringVar(&password, "password", "", "admin password (optional interactive)")
	createCmd.Flags().StringVar(&level, "level", "SUPERADMIN", "admin level: STAFF or SUPERADMIN")
	createCmd.MarkFlagRequired("username")
	createCmd.MarkFlagRequired("email")

	var getEmail string
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get admin by email",
		Run: func(cmd *cobra.Command, args []string) {
			if getEmail == "" {
				log.Fatalf("--email is required")
			}
			svc := newAdminService()
			a, err := svc.GetAdminByEmail(getEmail)
			if err != nil {
				log.Fatalf("admin not found: %v", err)
			}
			fmt.Printf("id=%s username=%s email=%s level=%s active=%v\n", a.ID, a.Username, a.Email, a.Level, a.IsActive)
		},
	}
	getCmd.Flags().StringVar(&getEmail, "email", "", "admin email (required)")

	var updEmailKey, updUsername, updEmail, updPassword, updLevel, updActiveStr string
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update an admin by email",
		Run: func(cmd *cobra.Command, args []string) {
			if updEmailKey == "" {
				log.Fatalf("--email is required")
			}
			svc := newAdminService()
			admin, err := svc.GetAdminByEmail(updEmailKey)
			if err != nil {
				log.Fatalf("admin not found: %v", err)
			}
			if updUsername != "" {
				admin.Username = updUsername
			}
			if updEmail != "" {
				admin.Email = updEmail
			}
			if updLevel != "" {
				admin.Level = updLevel
			}
			if updActiveStr != "" {
				active, err := strconv.ParseBool(updActiveStr)
				if err != nil {
					log.Fatalf("invalid --is_active value, use true/false")
				}
				admin.IsActive = active
			}
			if updPassword != "" {
				hashed, err := bcrypt.GenerateFromPassword([]byte(updPassword), bcrypt.DefaultCost)
				if err != nil {
					log.Fatalf("failed to hash password: %v", err)
				}
				admin.PasswordHash = string(hashed)
			}
			if err := svc.UpdateAdmin(admin); err != nil {
				log.Fatalf("failed to update admin: %v", err)
			}
			fmt.Printf("updated admin id=%s\n", admin.ID)
		},
	}
	updateCmd.Flags().StringVar(&updEmailKey, "email", "", "admin email (required, lookup key)")
	updateCmd.Flags().StringVar(&updUsername, "username", "", "new username")
	updateCmd.Flags().StringVar(&updEmail, "new-email", "", "new email")
	updateCmd.Flags().StringVar(&updPassword, "password", "", "new password")
	updateCmd.Flags().StringVar(&updLevel, "level", "", "new level")
	updateCmd.Flags().StringVar(&updActiveStr, "is_active", "", "set active state: true|false")

	var delEmail string
	var delYes bool
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an admin by email",
		Run: func(cmd *cobra.Command, args []string) {
			if delEmail == "" {
				log.Fatalf("--email is required")
			}
			if !delYes {
				var ans string
				fmt.Printf("Are you sure you want to delete admin email=%s? (y/N): ", delEmail)
				fmt.Scanln(&ans)
				if ans != "y" && ans != "Y" {
					fmt.Println("aborted")
					return
				}
			}
			svc := newAdminService()
			admin, err := svc.GetAdminByEmail(delEmail)
			if err != nil {
				log.Fatalf("admin not found: %v", err)
			}
			if err := svc.DeleteAdmin(admin.ID); err != nil {
				log.Fatalf("failed to delete admin: %v", err)
			}
			fmt.Printf("deleted admin id=%s email=%s\n", admin.ID, admin.Email)
		},
	}
	deleteCmd.Flags().StringVar(&delEmail, "email", "", "admin email (required)")
	deleteCmd.Flags().BoolVar(&delYes, "yes", false, "confirm deletion without prompt")

	adminCmd.AddCommand(createCmd, getCmd, updateCmd, deleteCmd)

	return []*cobra.Command{adminCmd}
}
