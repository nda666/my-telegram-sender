package telegram

import (
	"context"
	"fmt"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
)

type Contact struct {
	UserID     int64  `json:"user_id"`
	AccessHash int64  `json:"access_hash"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Username   string `json:"username"`
	Phone      string `json:"phone"`
}

// GetContacts mengambil semua kontak dari akun Telegram device.
func (s *Service) GetContacts(ctx context.Context, deviceID uint) ([]Contact, error) {
	var result []Contact
	err := s.Run(ctx, deviceID, func(ctx context.Context, client *telegram.Client, api *tg.Client) error {
		res, err := api.ContactsGetContacts(ctx, 0)
		if err != nil {
			return fmt.Errorf("contacts.getContacts: %w", err)
		}

		contacts, ok := res.(*tg.ContactsContacts)
		if !ok {
			// ContactsContactsNotModified — tidak ada perubahan
			return nil
		}

		userMap := make(map[int64]*tg.User, len(contacts.Users))
		for _, u := range contacts.Users {
			if user, ok := u.(*tg.User); ok {
				userMap[user.ID] = user
			}
		}

		for _, c := range contacts.Contacts {
			u, ok := userMap[c.UserID]
			if !ok {
				continue
			}
			result = append(result, Contact{
				UserID:     u.ID,
				AccessHash: u.AccessHash,
				FirstName:  u.FirstName,
				LastName:   u.LastName,
				Username:   u.Username,
				Phone:      u.Phone,
			})
		}
		return nil
	})
	return result, err
}

// ImportContact menambah satu kontak baru via nomor telepon.
func (s *Service) ImportContact(ctx context.Context, deviceID uint, phone, firstName, lastName string) (*Contact, error) {
	var contact *Contact
	err := s.Run(ctx, deviceID, func(ctx context.Context, client *telegram.Client, api *tg.Client) error {
		res, err := api.ContactsImportContacts(ctx, []tg.InputPhoneContact{
			{
				ClientID:  1,
				Phone:     phone,
				FirstName: firstName,
				LastName:  lastName,
			},
		})
		if err != nil {
			return fmt.Errorf("contacts.importContacts: %w", err)
		}
		if len(res.Users) == 0 {
			return fmt.Errorf("no user found for phone %s — pastikan nomor terdaftar di Telegram", phone)
		}
		u, ok := res.Users[0].(*tg.User)
		if !ok {
			return fmt.Errorf("unexpected user type")
		}
		contact = &Contact{
			UserID:     u.ID,
			AccessHash: u.AccessHash,
			FirstName:  u.FirstName,
			LastName:   u.LastName,
			Username:   u.Username,
			Phone:      u.Phone,
		}
		return nil
	})
	return contact, err
}

// EditContact mengupdate nama kontak — MTProto tidak punya endpoint edit,
// jadi import ulang dengan data baru (phone sama, nama beda).
func (s *Service) EditContact(ctx context.Context, deviceID uint, phone, firstName, lastName string) (*Contact, error) {
	return s.ImportContact(ctx, deviceID, phone, firstName, lastName)
}

// DeleteContact menghapus kontak berdasarkan userID + accessHash.
func (s *Service) DeleteContact(ctx context.Context, deviceID uint, userID, accessHash int64) error {
	return s.Run(ctx, deviceID, func(ctx context.Context, client *telegram.Client, api *tg.Client) error {
		_, err := api.ContactsDeleteContacts(ctx, []tg.InputUserClass{
			&tg.InputUser{
				UserID:     userID,
				AccessHash: accessHash,
			},
		})
		return err
	})
}
