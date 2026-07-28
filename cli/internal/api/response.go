package api

import (
	"net/http"
	"strings"
)

const maxPlainTextErrorRunes = 512

type responseStatus interface {
	GetStatusCode() int
	IsSuccessState() bool
}

type responseDetails interface {
	responseStatus
	GetContentType() string
	String() string
}

func errorFromResponse(resp responseStatus, craterCode int, msg string) error {
	status := resp.GetStatusCode()
	if !resp.IsSuccessState() {
		return &RequestError{
			HTTPStatus: status,
			CraterCode: craterCode,
			Msg:        responseErrorMessage(resp, msg),
		}
	}
	if craterCode != 0 {
		return &RequestError{
			HTTPStatus: status,
			CraterCode: craterCode,
			Msg:        msg,
		}
	}
	return nil
}

func responseErrorMessage(resp responseStatus, msg string) string {
	if msg = strings.TrimSpace(msg); msg != "" {
		return msg
	}
	if detailed, ok := resp.(responseDetails); ok && strings.HasPrefix(strings.ToLower(detailed.GetContentType()), "text/plain") {
		plainText := strings.Join(strings.Fields(detailed.String()), " ")
		if runes := []rune(plainText); len(runes) > maxPlainTextErrorRunes {
			plainText = string(runes[:maxPlainTextErrorRunes]) + "..."
		}
		if plainText != "" {
			return plainText
		}
	}
	return http.StatusText(resp.GetStatusCode())
}
