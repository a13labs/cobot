package consoleChannel

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/a13labs/cobot/internal/agent"
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

// StartBot initializes and starts the bot with a channel listener.
func Start() {

	agent.SetWriterFunc(func(text string) error { fmt.Println(text); return nil })

	fn := func(userInput string) error {
		agent.DispatchInput(userInput)
		return nil
	}

	agent.SayHello()
	err := forEachInput(os.Stdin, fn)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	agent.SayGoodBye()
}
