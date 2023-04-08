package components

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

func AskAYesNoQuestion(msg string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [(y)es/(n)o]: ", msg)

		answer, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal().Err(err).Msg("failed to read response")
		}

		answer = strings.ToLower(strings.TrimSpace(answer))

		if answer == "y" || answer == "yes" {
			return true
		} else if answer == "n" || answer == "no" {
			return false
		}
	}
}
