package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"net/http"
	"time"

	"github.com/anthonynsimon/bild"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"golang.org/x/image/webp"
)

func DownloadFile(bot *tgbotapi.BotAPI, fileid string) (*bytes.Buffer, error) {
	client := http.Client{
		Timeout: 60 * time.Second,
	}
	link, err := bot.GetFileDirectURL(fileid)
	if err != nil {
		return nil, err
	}
	resp, err := client.Get(link)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", resp.Status)
	}
	buf := &bytes.Buffer{}
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}
	return buf, resp.Body.Close()
}

func ProcessSticker(bot *tgbotapi.BotAPI, message tgbotapi.Message) {
	buf, err := DownloadFile(bot, message.Sticker.FileID)
	if err != nil {
		log.Printf("error while downloading sticker: %v\n", err)
		return
	}
	img, err := webp.Decode(buf)
	if err != nil {
		log.Printf("error while decoding sticker: %v\n", err)
		return
	}
	buf.Reset()
	imgBuf := buf
	err = png.Encode(imgBuf, img)
	if err != nil {
		log.Printf("error while encoding png image: %v\n", err)
		return
	}

	_, err = bot.Send(tgbotapi.NewDocumentUpload(message.Chat.ID,
		tgbotapi.FileBytes{
			Name:  message.Sticker.Emoji + ".png",
			Bytes: imgBuf.Bytes(),
		}))
	if err != nil {
		log.Printf("error while sending image: %v\n", err)
	}

	imgBuf.Reset()
	photoBuf := buf
	photo := bild.NormalBlend(bild.Crop(image.White, image.Rect(0, 0, message.Sticker.Width, message.Sticker.Height)), img)
	err = jpeg.Encode(photoBuf, photo, nil)
	if err != nil {
		log.Printf("error while encoding jpeg image: %v\n", err)
		return
	}
	_, err = bot.Send(tgbotapi.NewPhotoUpload(message.Chat.ID,
		tgbotapi.FileBytes{
			Name:  message.Sticker.Emoji + ".jpeg",
			Bytes: photoBuf.Bytes(),
		}))
	if err != nil {
		log.Printf("error while sending image: %v\n", err)
	}
}
