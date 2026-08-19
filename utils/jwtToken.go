package utils

import (
	"errors"
	"fmt"
	"time"

	"ChatApp/config"

	"github.com/golang-jwt/jwt/v5"
)

// Claims 载荷
type Claims struct {
	UID string `json:"uid"`
	jwt.RegisteredClaims
}

// GenJwtToken 根据用户ID生成token
func GenJwtToken(UID string) (string, error) {
	// 从全局配置读取密钥
	secret := config.Conf.Jwt.Secret
	if secret == "" {
		return "", errors.New("未配置JWT密钥")
	}

	// 解析配置里的过期时长字符串
	dur, err := time.ParseDuration(config.Conf.Jwt.Expire)
	if err != nil {
		return "", errors.New("jwt过期时间配置格式错误，示例：168h、2h30m")
	}

	claims := Claims{
		UID: UID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(dur)), //过期时间
			IssuedAt:  jwt.NewNumericDate(time.Now()),          //签发时间
			NotBefore: jwt.NewNumericDate(time.Now()),          //生效时间 这些时间用的都是jwt.NewNumericDare配合time模块设置的
			Issuer:    "ChatApp",                               //签发者 发行方
			Subject:   "token",                                 //主题
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims) //创建令牌对象，参数1是加密方法，参数2是需要加密的数据包

	// 原理：
	// 1. 将header和payload转换为JSON并进行Base64编码
	//	  header包含令牌类型（typ: "JWT"）和签名算法（如 HS256、RS256 等）
	//    payload存储需要传递的用户信息
	//    signature用于验证令牌的完整性和真实性，防止被篡改
	// 2. 用指定算法和密钥对"编码后的header.编码后的payload"进行加密
	// 3. 生成的加密结果就是签名，最终令牌格式为"header.payload.signature"
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

// ParseJwtToken 解析token
func ParseJwtToken(tokenStr string) (string, error) {
	secret := config.Conf.Jwt.Secret
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{}, // 空结构体，用于接收解析后的数据
		func(token *jwt.Token) (interface{}, error) {
			//验证签名算法
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("非法签名算法：%v", token.Header["alg"])
			}
			return []byte(secret), nil
		})
	if err != nil {
		return "", errors.New("token无效或已过期")
	}
	//验证token有效性并提取claims
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims.UID, nil
	}
	return "", errors.New("token解析失败")
}
