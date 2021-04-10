package service

import (
	"encoding/hex"
	"fmt"
	"nspyf/model"
	"nspyf/model/dao"
	"nspyf/model/dto"
	"nspyf/util"
	"strconv"
)

func Register(req *dto.Register) uint {
	code := CheckUsername(req.Username)
	if code != 0 {
		return code
	}
	code = CheckPassword(req.Password)
	if code != 0 {
		return code
	}

	user := &dao.User{
		Username: req.Username,
	}
	_ = user.Retrieve()
	if user.ID != 0 {
		return 6
	}

	salt, err := util.RandHexStr(64)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	password := hex.EncodeToString(util.SHA512([]byte(req.Password + salt)))

	user = &dao.User{
		Username:    req.Username,
		Password:    password,
		Salt:        salt,
		LoginStatus: "0",
	}
	err = user.Create()
	if err != nil {
		fmt.Println(err)
		return 1
	}
	return 0
}

func Login(req *dto.Login) (*map[string]interface{}, uint) {
	isEmail := false
	length := len(req.User)
	for i := 0; i < length; i++ {
		if req.User[i] == '@' {
			isEmail = true
			break
		}
	}

	user := &dao.User{}
	if isEmail {
		user.Email = req.User
	} else {
		user.Username = req.User
	}
	_ = user.Retrieve()
	if user.ID == 0 {
		return nil, 7
	}

	password := hex.EncodeToString(util.SHA512([]byte(req.Password + user.Salt)))
	if user.Password != password {
		return nil, 7
	}

	token, err := model.Jwt.GenerateToken(strconv.Itoa(int(user.ID)), user.LoginStatus)
	if err != nil {
		fmt.Println(err)
		return nil, 1
	}

	data := &map[string]interface{}{
		"id":       user.ID,
		"token":    token,
		"username": user.Username,
	}

	return data, 0
}

func CheckUsername(username string) uint {
	usernameLen := len(username)
	if usernameLen < 2 || usernameLen > 16 {
		return 4
	}
	for i := 0; i < usernameLen; i++ {
		if (username[i] < 'a' || 'z' < username[i]) && (username[i] < 'A' || 'Z' < username[i]) && (username[i] < '0' || '9' < username[i]) {
			return 4
		}
	}
	return 0
}

func CheckPassword(password string) uint {
	passwordLen := len(password)
	if passwordLen < 8 || passwordLen > 32 {
		return 5
	}
	//[33,126]覆盖了大小写字母、数字、普通可见符号
	for i := 0; i < passwordLen; i++ {
		if password[i] < 33 || password[i] > 126 {
			return 5
		}
	}
	return 0
}

func SetPassword(req *dto.SetPassword, id uint) uint {
	code := CheckPassword(req.NewPassword)
	if code != 0 {
		return code
	}

	user := &dao.User{
		ID: id,
	}
	err := user.Retrieve()
	if err != nil {
		return 1
	}

	shaPassword := hex.EncodeToString(util.SHA512([]byte(req.Password + user.Salt)))
	if user.Password != shaPassword {
		return 14
	}

	return updatePassword(req.NewPassword, id)
}

//更新盐、个人登录状态、密码
func updatePassword(newPassword string, id uint) uint {
	saltStr, err := util.RandHexStr(64)
	if err != nil {
		fmt.Println(err)
		return 1
	}
	shaNewPassword := hex.EncodeToString(util.SHA512([]byte(newPassword + saltStr)))
	loginStatus, err := util.RandHexStr(8)
	if err != nil {
		fmt.Println(err)
		return 1
	}

	user := &dao.User{
		ID: id,
	}
	err = user.Update(map[string]interface{}{
		"Password":    shaNewPassword,
		"Salt":        saltStr,
		"LoginStatus": loginStatus,
	})
	if err != nil {
		fmt.Println(err)
		return 1
	}

	//删用户缓存
	profile := &dao.UserProfile{
		ID: id,
	}
	err = profile.DelCache()
	if err != nil {
		fmt.Println(err)
	}

	return 0
}
