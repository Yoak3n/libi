package service

import (
	"github.com/Yoak3n/libi/shared/domain/model/table"
)

func CreateSignedUser(uid uint, description string) error {
	return SignedUserRepo.CreateSignedUser(&table.SignedUserTable{
		UID:         uid,
		Description: description,
	})
}

func GetSignedUser(uid uint) (*table.SignedUserTable, error) {
	return SignedUserRepo.ReadSignedUser(uid)
}

func GetAllSignedUsers() ([]*table.SignedUserTable, error) {
	return SignedUserRepo.ReadAllSignedUsers()
}

func UpdateSignedUserDescription(uid uint, description string) error {
	user, err := SignedUserRepo.ReadSignedUser(uid)
	if err != nil {
		return err
	}
	user.Description = description
	return SignedUserRepo.UpdateSignedUser(user)
}

func DeleteSignedUser(uid uint) error {
	return SignedUserRepo.DeleteSignedUser(uid)
}
