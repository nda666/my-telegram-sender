package utils

import "strings"

func NormalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)

	if IsLocalPhone(phone) {
		return LocalToE164(phone)
	}

	return phone
}

func IsLocalPhone(phone string) bool {
	return strings.HasPrefix(phone, "08")
}

func LocalToE164(phone string) string {
	return "+62" + phone[1:]
}
