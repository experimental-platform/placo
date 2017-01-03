package update

import "os"

type buttonState string

const (
	buttonBusy     buttonState = "b 1000"
	buttonWarning              = "w 400"
	buttonError                = "e 400"
	buttonNoise                = "n 1000"
	buttonShimmer              = "s 1000"
	buttonRainbow              = "r 1000"
	buttonPower                = "p 700"
	buttonHDD                  = "h 700"
	buttonStartup              = "u 700"
	buttonShutdown             = "d 700"
)

func button(state buttonState) error {
	buttonPath := "/dev/protobutton0"
	_, err := os.Stat(buttonPath)
	if err != nil {
		return err
	}

	buttonDev, err := os.OpenFile(buttonPath, os.O_TRUNC|os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer buttonDev.Close()

	_, err = buttonDev.WriteString(string(state) + "\n")
	if err != nil {
		return err
	}

	err = buttonDev.Sync()
	if err != nil {
		return err
	}

	return err
}
