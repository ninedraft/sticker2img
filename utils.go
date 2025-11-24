package main

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
	"github.com/anthonynsimon/bild/blend"
	"github.com/anthonynsimon/bild/transform"
	"golang.org/x/image/webp"
)

const maxSize = 1024 * 1024

var errTooBig = errors.New("file is too big")

func DownloadFile(bot *tgbotapi.BotAPI, fileid string) ([]byte, error) {
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", resp.Status)
	}

	re := &io.LimitedReader{
		R: resp.Body,
		N: maxSize + 1,
	}

	body, err := io.ReadAll(re)

	if re.N == 0 {
		return nil, errTooBig
	}

	return body, err
}

func ProcessSticker(bot *tgbotapi.BotAPI, message tgbotapi.Message) {
	defer func() {
		err := recover()
		if err != nil {
			log.Printf("panic: %v\n", err)
		}
	}()

	body, err := DownloadFile(bot, message.Sticker.FileID)
	if err != nil {
		log.Printf("error while downloading sticker: %v\n", err)
		return
	}
	img, err := webp.Decode(bytes.NewReader(body))

	if errors.Is(err, errTooBig) {
		log.Printf("error while decoding sticker: %v\n", err)

		_, _ = bot.Send(tgbotapi.NewMessage(message.Chat.ID, err.Error()))
		return
	}

	if err != nil {
		log.Printf("error while decoding sticker: %v\n", err)

		_, _ = bot.Send(tgbotapi.NewMessage(message.Chat.ID, "unsuppored file format "+http.DetectContentType(body)))
		return
	}

	filename := fmt.Sprintf("%s_%s", message.Sticker.Emoji, strings.ToLower(rand.Text()))

	{
		imgBuf := &bytes.Buffer{}
		err = png.Encode(imgBuf, img)
		if err != nil {
			log.Printf("error while encoding png image: %v\n", err)
			return
		}

		document := tgbotapi.NewDocument(message.Chat.ID,
			tgbotapi.FileBytes{
				Name:  filename + ".png",
				Bytes: imgBuf.Bytes(),
			})

		document.ReplyParameters.ChatID = message.Chat.ID
		document.ReplyParameters.MessageID = message.MessageID

		_, err = bot.Send(document)
		if err != nil {
			log.Printf("error while sending image: %v\n", err)
		}
	}

	imgs := map[string][]byte{}
	photo := blend.Normal(transform.Crop(whiteImg, image.Rect(0, 0, message.Sticker.Width, message.Sticker.Height)), img)

	func() {
		buf := &bytes.Buffer{}
		err = jpeg.Encode(buf, photo, jpegOpts)
		if err != nil {
			log.Printf("error while encoding jpeg image: %v\n", err)
			return
		}

		imgs[filename+".jpg"] = buf.Bytes()
	}()

	func() {
		buf := &bytes.Buffer{}
		photo := transform.Resize(photo, message.Sticker.Width*3/7, message.Sticker.Height*3/7, transform.Linear)
		err = jpeg.Encode(buf, photo, jpegOpts)
		if err != nil {
			log.Printf("error while encoding jpeg image: %v\n", err)
			return
		}

		imgs[filename+"_small.jpg"] = buf.Bytes()
	}()

	_, err = bot.SendMediaGroup(composeAlbum(message.Chat.ID, message.MessageID, imgs))
	if err != nil {
		log.Print("sending photos", "error", err)
	}
}

var jpegOpts = &jpeg.Options{
	Quality: 100,
}

func composeAlbum(chatID int64, msgID int, imgs map[string][]byte) tgbotapi.MediaGroupConfig {
	files := make([]tgbotapi.InputMedia, 0, len(imgs))
	filenames := slices.Sorted(maps.Keys(imgs))

	for _, filename := range filenames {
		file := imgs[filename]

		photo := tgbotapi.NewInputMediaPhoto(tgbotapi.FileBytes{
			Name:  filename,
			Bytes: file,
		})
		photo.Caption = filename
		files = append(files, &photo)
	}

	album := tgbotapi.NewMediaGroup(chatID, files)
	album.ReplyParameters.MessageID = msgID
	return album
}
