package convert

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

type SSEEvent struct {
	Event string
	Data  string
}

func ParseSSE(r io.Reader) (<-chan SSEEvent, <-chan error) {
	events := make(chan SSEEvent)
	errs := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errs)

		scanner := bufio.NewScanner(r)
		var eventType string
		var dataLines []string

		for scanner.Scan() {
			line := scanner.Text()

			if line == "" {
				if len(dataLines) > 0 {
					data := strings.Join(dataLines, "\n")
					events <- SSEEvent{Event: eventType, Data: data}
				}
				eventType = ""
				dataLines = nil
				continue
			}

			if strings.HasPrefix(line, ":") {
				continue
			}

			if strings.HasPrefix(line, "event: ") {
				eventType = strings.TrimPrefix(line, "event: ")
				continue
			}

			if strings.HasPrefix(line, "data: ") {
				dataLines = append(dataLines, strings.TrimPrefix(line, "data: "))
				continue
			}
		}

		if scanner.Err() != nil {
			errs <- scanner.Err()
		}
	}()

	return events, errs
}

func WriteSSE(w io.Writer, event SSEEvent) error {
	if event.Event != "" {
		if _, err := io.WriteString(w, "event: "+event.Event+"\n"); err != nil {
			return err
		}
	}

	if event.Data == "[DONE]" {
		_, err := io.WriteString(w, "data: [DONE]\n\n")
		return err
	}

	if _, err := io.WriteString(w, "data: "+event.Data+"\n\n"); err != nil {
		return err
	}

	return nil
}

func writeSSEJSON(w io.Writer, eventType string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return WriteSSE(w, SSEEvent{Event: eventType, Data: string(data)})
}
