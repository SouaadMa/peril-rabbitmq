package gamelogic

import "context"

func InputLines(ctx context.Context) <-chan []string {
	lines := make(chan []string)
	go func() {
		defer close(lines)
		for {
			words := GetInput()
			if words == nil {
				return
			}
			if len(words) == 0 {
				continue
			}
			select {
			case lines <- words:
			case <-ctx.Done():
				return
			}
		}
	}()
	return lines
}
