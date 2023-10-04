package consoleBot

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/a13labs/infrabot/internal/agent"
)

// ForEachInput calls the given callback function for each line of input.
func forEachInput(r io.Reader, callback func(text string) error) error {
	scanner := bufio.NewScanner(r)
	for {
		fmt.Print("> ")
		scanner.Scan()
		text := scanner.Text()
		if text == "" {
			break
		}
		if err := callback(text); err != nil {
			return err
		}
	}
	return nil
}

// StartBot initializes and starts the Telegram bot with a channel listener.
func Start() {
	fn := func(userInput string) error {
		msg, err := agent.RunAction(userInput)
		if err != nil {
			return err
		}
		fmt.Println(msg)
		return nil
	}

	fmt.Println(agent.SayHello())
	fmt.Println("To interact with me write your queries/actions. To quit just enter an empty line!")
	err := forEachInput(os.Stdin, fn)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	fmt.Println(agent.SayGoodBye())
}
