package handler

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"
)

const maxCoAuthoredByEmailLength = 254

func parseCoAuthoredByEmail(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	if strings.ContainsAny(trimmed, "\r\n") {
		return "", fmt.Errorf("co_authored_by_email must be a single-line email address")
	}
	if utf8.RuneCountInString(trimmed) > maxCoAuthoredByEmailLength {
		return "", fmt.Errorf("co_authored_by_email must be %d characters or fewer", maxCoAuthoredByEmailLength)
	}
	addr, err := mail.ParseAddress(trimmed)
	if err != nil {
		return "", fmt.Errorf("co_authored_by_email must be a valid email address")
	}
	email := strings.ToLower(strings.TrimSpace(addr.Address))
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return "", fmt.Errorf("co_authored_by_email must be a valid email address")
	}
	if strings.ContainsAny(email, "<>") {
		return "", fmt.Errorf("co_authored_by_email must be a valid email address")
	}
	return email, nil
}
