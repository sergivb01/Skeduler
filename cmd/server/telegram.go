package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

type telegramClient struct {
	Token  string `yaml:"token" json:"token"`
	ChatId int64  `yaml:"chat_id" json:"chat_id"`
}

func (t *telegramClient) sendNotification(job jobs.Job) error {
	bot, err := tgbotapi.NewBotAPI(t.Token)
	if err != nil {
		return fmt.Errorf("error creating bot: %w", err)
	}

	fileName := fmt.Sprintf("./logs/%v.log", job.ID)
	b, err := ioutil.ReadFile(fileName)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("error opening log file: %w", err)
	}

	message := tgbotapi.DocumentConfig{
		BaseFile: tgbotapi.BaseFile{
			BaseChat: tgbotapi.BaseChat{
				ChatID: t.ChatId,
			},
			File: tgbotapi.FileBytes{
				Name:  fileName,
				Bytes: b,
			},
		},
		Caption: fmt.Sprintf(`
		Experiment actualitzat:
		 * ID: %v
			* Nom: %s
			* Descripció: %s
			* Imatge: %s
			* Data creació: %s
			* Data actualització: %s
			* Estatus: %s
		`, job.ID, job.Name, job.Description, job.Docker.Image, job.CreatedAt, job.UpdatedAt, job.Status),
	}

	_, err = bot.Send(message)
	return err
}
