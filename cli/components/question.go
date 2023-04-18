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

	fmt.Printf("%s [Y/n] ", msg)

	answer, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read response")
	}

	answer = strings.ToLower(strings.TrimSpace(answer))

	return answer == "y" || answer == "yes"
}
