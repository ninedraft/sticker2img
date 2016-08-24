package main

import (
	"flag"
	"image"
	"image/color"
	"log"
	"runtime"

	"github.com/go-telegram-bot-api/telegram-bot-api"
)

func init() {
	whitePage := image.NewRGBA(image.Rect(0, 0, 512, 512))
	for x := 0; x < 512; x++ {
		for y := 0; y < 512; y++ {
			whitePage.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	BackGroundImage = whitePage
}

func main() {
	Token := flag.String("token", "", "Telegram Bot API token")
	debug := flag.Bool("debug", false, "show debug information")
	flag.Parse()
	if *Token == "" {
		log.Fatal("Token flag required!")
	}
	bot, err := tgbotapi.NewBotAPI(*Token)
	if err != nil {
		log.Fatalln(err)
	}
	bot.Debug = *debug
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.Printf("Running on %d CPU\n", runtime.NumCPU())
	log.Printf("Authorized on account %q\n", bot.Self.UserName)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 3600

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatalf("error while getting update chan: %v\n", err)
	}
	for update := range updates {
		if update.Message == nil {
			continue
		}
		if update.Message.Sticker != nil {
			log.Printf("@%s: %s\n", update.Message.From.UserName, update.Message.Sticker.Emoji)
			go ProcessSticker(bot, *update.Message)
		}
		if update.Message.Command() == "start" {
			go func() {
				repl := tgbotapi.NewMessage(update.Message.Chat.ID, "Hi!\nSend me a sticker and I'll return you a photo and a PNG image!")
				repl.ReplyToMessageID = update.Message.MessageID
				_, err := bot.Send(repl)
				if err != nil {
					log.Printf("error while sending 'hello' message: %v\n", err)
				}
			}()
		}
	}
}
