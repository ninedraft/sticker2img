package main

import (
	"bytes"
	"flag"
	"image"
	"image/png"
	"io"
	"log"
	"os"

	"github.com/eternnoir/gotelebot"
	"golang.org/x/image/webp"
)

func main() {
	Token := flag.String("token", "", "Telegram Bot API token")
	bot := gotelebot.InitTeleBot(*Token)
	go func(b *gotelebot.TeleBot) {
		err := b.StartPolling(true, 120)
		if err != nil {
			log.Fatalln(err)
		}
	}(bot)
	var err error
	var data *[]byte
	var img image.Image
	var file *os.File
	buf := &bytes.Buffer{}
	for message := range bot.Messages {
		if message.Sticker != nil {
			log.Printf("@%s -> %s\n", message.From.Username, message.Sticker.FileId)
			data, err = bot.DownloadFile(message.Sticker.FileId)
			if err != nil {
				log.Printf("error while downloading sticker: %v\n", err)
				continue
			}
			img, err = webp.Decode(bytes.NewReader(*data))
			if err != nil {
				log.Printf("error while decoding sticker: %v\n", err)
				continue
			}
			buf.Reset()
			err = png.Encode(buf, img)
			if err != nil {
				log.Printf("error while encoding png image: %v\n", err)
				continue
			}
			file, err = os.Create(message.Sticker.FileId + ".png")
			if err != nil {
				log.Printf("error while creating file %q: %v\n", message.Sticker.FileId+".png", err)
				continue
			}
			_, err = io.Copy(file, bytes.NewReader(buf.Bytes()))
			if err != nil {
				log.Printf("error while writing file %q: %v\n", message.Sticker.FileId+".png", err)
				continue
			}
			file.Close()
			bot.SendDocument(int(message.Chat.Id), "")
			/*
				            _, err = bot.SendDocument(int(message.Chat.Id), buf.String(), nil)
							if err != nil {
								log.Printf("error while uploading png image: %v\n", err)
								continue
							}
			*/
		}
	}
}
