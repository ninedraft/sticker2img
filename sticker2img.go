package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"

	_ "embed"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
	"golang.org/x/sync/errgroup"
)

var whiteImg = func() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 512, 512))
	white := color.RGBA{255, 255, 255, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{white}, image.Point{}, draw.Src)

	return img
}()

type config struct {
	token   string
	log     *slog.Logger
	maxJobs int
	debug   bool
}

func run(cfg config) (int, error) {
	log := cfg.log

	bot, err := tgbotapi.NewBotAPI(cfg.token)
	if err != nil {
		return 1, fmt.Errorf("creating bot: %w", err)
	}
	bot.Debug = cfg.debug

	log.Info("Authorized on account", "bot", bot.Self.UserName)

	ctx := context.Background()

	wg := &errgroup.Group{}
	wg.SetLimit(cfg.maxJobs)

	updateParams := tgbotapi.UpdateConfig{}

	handleUpdates := func(updates []tgbotapi.Update) error {
		for _, update := range updates {
			if update.Message == nil || !update.Message.Chat.IsPrivate() {
				continue
			}

			if update.Message.Sticker != nil {
				log.Info("new message",
					"from", update.Message.From.UserName,
					"emoji", update.Message.Sticker.Emoji)
				wg.Go(func() error {
					ProcessSticker(bot, *update.Message)
					return nil
				})
			}

			if update.Message.Command() == "start" {
				wg.Go(func() error {
					repl := tgbotapi.NewMessage(update.Message.Chat.ID, "Hi!\nSend me a sticker and I'll return you a photo and a PNG image!")
					repl.ReplyParameters.MessageID = update.Message.MessageID
					_, err := bot.Send(repl)
					if err != nil {
						log.Warn("error while sending 'hello' message", "error", err)
					}

					return nil
				})
			}
		}

		return nil
	}

	for retry := 0; ; {
		updates, err := bot.GetUpdatesWithContext(ctx, updateParams)
		if err != nil {
			log.Warn("requesting updates", "error", err)
			continue
		}
		if len(updates) == 0 {
			continue
		}

		updateParams.Offset = updates[len(updates)-1].UpdateID + 1

		if err := handleUpdates(updates); err != nil {
			retry++
			log.Error("handling updates", "error", err)
			time.Sleep(10 * time.Second)
		}
		if retry >= 3 {
			return 1, err
		}
	}
}

//go:embed usage.txt
var usage string

func main() {
	cfg := config{
		maxJobs: runtime.NumCPU(),
	}

	tokenFile := "telegram-token.txt"
	flag.StringVar(&tokenFile, "token-file", tokenFile, "file with telegram token secret. Required")
	flag.IntVar(&cfg.maxJobs, "jobs", cfg.maxJobs, "max concurrent jobs, <= 0 means default")

	logLevel := slog.LevelInfo
	flag.TextVar(&logLevel, "level", logLevel, "log level")

	flag.Usage = func() {
		output := flag.CommandLine.Output()

		fmt.Fprintf(output, "%s\n\n", usage)
		flag.PrintDefaults()
	}

	flag.Parse()

	cfg.debug = logLevel <= slog.LevelDebug

	if cfg.maxJobs <= 0 {
		cfg.maxJobs = runtime.NumCPU()
	}

	if tokenFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	tokenData, err := os.ReadFile(tokenFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to read token file %q: %v", tokenFile, err)
		flag.Usage()
		os.Exit(1)
	}

	cfg.token = strings.TrimSpace(string(tokenData))

	cfg.log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	status, err := run(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n\nERROR: %v\n\n", err)
		status = max(status, 1)
	}

	if status != 0 {
		os.Exit(status)
	}
}
