package utils

import (
	"fmt"
	"log/slog"
	"sync"
)

func BackgroundTask(wg *sync.WaitGroup, fn func() error) {
	wg.Add(1)

	go func() {
		defer wg.Done()

		defer func() {
			err := recover()
			if err != nil {
				slog.Error("[backgroundTask]", slog.String("error", fmt.Sprint(err)))
			}
		}()

		err := fn()
		if err != nil {
			slog.Error("[backgroundTask]", slog.String("error", fmt.Sprint(err)))
		}
	}()
}
