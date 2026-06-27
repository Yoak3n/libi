package main

import (
	"context"
	"fmt"
	"os"
	"time"

	sharedconfig "github.com/Yoak3n/libi/shared/config"
	"github.com/Yoak3n/libi/shared/domain/model/table"
	"github.com/Yoak3n/libi/shared/login"
	"github.com/mdp/qrterminal/v3"
	"github.com/urfave/cli/v3"
	"troll/service"
)

type configArgs struct {
	Proxy string
	List  bool
	Clean bool
}

func configCommand() *cli.Command {
	c := &configArgs{}
	return &cli.Command{
		Name:  "config",
		Usage: "manage auth and settings",
		Commands: []*cli.Command{
			configLoginCommand(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			ensureService()
			if c.Clean {
				return cleanAccounts()
			}
			if c.List {
				return listAccounts()
			}
			if c.Proxy != "" {
				return setProxy(c.Proxy)
			}
			return cli.Exit("Use --proxy, --list, --clean, or 'config login'", 1)
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "proxy",
				Usage:       "set HTTP proxy",
				Aliases:     []string{"p"},
				Destination: &c.Proxy,
			},
			&cli.BoolFlag{
				Name:        "list",
				Usage:       "list stored accounts",
				Aliases:     []string{"l"},
				Destination: &c.List,
			},
			&cli.BoolFlag{
				Name:        "clean",
				Usage:       "validate all accounts, refresh invalid ones, remove dead ones",
				Destination: &c.Clean,
			},
		},
	}
}

func configLoginCommand() *cli.Command {
	return &cli.Command{
		Name:  "login",
		Usage: "scan QR code to login and store cookie + refresh token",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runQRLogin()
		},
	}
}

func runQRLogin() error {
	url, key, err := login.GenerateQRCode()
	if err != nil {
		return fmt.Errorf("generate QR code: %w", err)
	}

	fmt.Println("Scan this QR code with the Bilibili app:")
	qrterminal.GenerateWithConfig(url, qrterminal.Config{
		Level:     qrterminal.L,
		Writer:    os.Stdout,
		BlackChar: qrterminal.BLACK,
		WhiteChar: qrterminal.WHITE,
	})
	fmt.Println("Waiting for scan...")

	client := login.NewClient("", "", func(cookie, refreshToken string) {
		if sharedconfig.Conf != nil && sharedconfig.Conf.Auth != nil {
			sharedconfig.Conf.Auth.AddAccount(cookie, refreshToken)
			sharedconfig.SaveAuth()
		}
		uid := sharedconfig.ExtractUID(cookie)
		fmt.Println()
		fmt.Println("Login successful!")
		if uid != 0 {
			fmt.Printf("  UID: %d\n", uid)
		}
		fmt.Printf("  Cookie: %s...\n", cookie[:min(40, len(cookie))])
		fmt.Println("  Refresh token saved (auto-renewal enabled)")
	})

	done := make(chan error, 1)
	go func() {
		done <- client.PollLogin(key)
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		return nil
	case <-time.After(3 * time.Minute):
		return fmt.Errorf("login timed out after 3 minutes")
	}
}

func setProxy(proxy string) error {
	if sharedconfig.Conf != nil {
		sharedconfig.Conf.Proxy = proxy
	}
	// Also save to DB if available
	if service.ConfRepo != nil {
		existing, err := service.ConfRepo.ReadConfigurationByType("proxy")
		if err != nil || existing == nil {
			service.ConfRepo.CreateConfiguration(&table.ConfigurationTable{
				Type: "proxy",
				Data: proxy,
			})
		} else {
			existing.Data = proxy
			service.ConfRepo.UpdateConfiguration(existing)
		}
	}
	fmt.Printf("Proxy set to: %s\n", proxy)
	return nil
}

func listAccounts() error {
	sharedconfig.Reload()
	if sharedconfig.Conf == nil || sharedconfig.Conf.Auth == nil {
		fmt.Println("No config loaded")
		return nil
	}
	accounts := sharedconfig.Conf.Auth.Accounts
	if len(accounts) == 0 {
		fmt.Println("No accounts stored. Use 'troll config login' to add one.")
		return nil
	}
	rows := make([][]string, 0, len(accounts))
	for i, acc := range accounts {
		uid := "-"
		if acc.UID != 0 {
			uid = fmt.Sprintf("%d", acc.UID)
		}
		cookie := acc.Cookie
		if len(cookie) > 40 {
			cookie = cookie[:40] + "..."
		}
		renew := "no"
		if acc.RefreshToken != "" {
			renew = "yes"
		}
		rows = append(rows, []string{fmt.Sprintf("%d", i), uid, cookie, renew})
	}
	printTable([]string{"#", "UID", "Cookie", "Auto-renew"}, rows)
	return nil
}

func cleanAccounts() error {
	if sharedconfig.Conf == nil || sharedconfig.Conf.Auth == nil {
		return fmt.Errorf("no config loaded")
	}
	fmt.Println("Validating all accounts...")
	before := len(sharedconfig.Conf.Auth.Accounts)
	removed := sharedconfig.Conf.Auth.EnsureAccounts()
	if removed > 0 {
		fmt.Printf("  Removed %d dead accounts (%d -> %d)\n", removed, before, before-removed)
	} else {
		fmt.Printf("  All %d accounts valid\n", before)
	}
	return nil
}
