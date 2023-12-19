// Package auth provides generic authentication functionality.
//
// Note this is not 100% secure and should only be used for prototyping,
// not for production systems or systems that are accessed by real users.
package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/nkhoaa96/go-base/cache"
	"github.com/nkhoaa96/go-base/storage/local"
	"golang.org/x/crypto/bcrypt"
	"strconv"
	"time"
)

var (
	authCache cache.Cacher // cache is used for storing authentication tokens.
	//logger         = log.NewLogger(".logs/app.log")
	//accountAuth, _ = secretmanage.GetAccountAuth()
)

const (
	ADMIN = "admin"
	STAFF = "staff"
)

type AccountAuth struct {
	OmsHostAddr      string
	OmsUserName      string
	OmsUserPassword  string
	EsbHostAddr      string
	EsbUserName      string
	EsbPassword      string
	LdapUrlAddress   string
	ApimHostAddress  string
	ApimUserName     string
	ApimUserPassword string
	ApimUserAuth     string
	EkycHost         string
	EkycAccessKey    string
	EkycSecret       string
}

// admin is a staff
type User struct {
	Id                   string    `json:"id"`
	FullName             string    `json:"full_name"`
	UserName             string    `json:"user_name"`
	TaxCode              string    `json:"tax_code"`
	BankAccount          string    `json:"bank_account"`
	BankCode             string    `json:"bank_code"`
	Phone                string    `json:"phone_num"`
	Email                string    `json:"email"`
	Gender               string    `json:"gender"`
	Avatar               string    `json:"avatar"`
	Status               string    `json:"status"`
	IdcardNum            string    `json:"idcard_num"`
	ShowroomId           string    `json:"showroom_id"`
	IdcardIssueDate      string    `json:"idcard_issue_date"`
	IdcardIssuePlaceName string    `json:"idcard_issue_place_name"`
	IdcardIssuePlaceCode string    `json:"idcard_issue_place_code"`
	DateOfBirth          string    `json:"idcard_issue_date"`
	VibStaffId           string    `json:"-"`
	Role                 string    `json:"role"`
	CreateAt             time.Time `json:"-"`
	UpdateAt             time.Time `json:"-"`
}

type OMSInfo struct {
	ID                   string `json:"id"`
	Loginname            string `json:"loginName"`
	Fullname             string `json:"fullName"`
	Fullnameen           string `json:"fullNameEn"`
	Email                string `json:"email"`
	Code                 string `json:"code"`
	Address              string `json:"address"`
	Addressid            string `json:"addressId"`
	Mobile               string `json:"mobile"`
	Status               string `json:"status"`
	Ext                  string `json:"ext"`
	Imageurl             string `json:"imageUrl"`
	Joiningdate          string `json:"joiningDate"`
	StartDate            string `json:"contractStartDate"`
	Gender               string `json:"gender"`
	Birthday             string `json:"birthDay"`
	LineManagerLoginName string `json:"lineManagerLoginName"`
	LineManagerEmail     string `json:"lineManagerEmail"`
	Deptid               string `json:"deptId"`
	Deptcode             string `json:"deptCode"`
	Idno                 string `json:"idNo"`
	Idissuedate          string `json:"idIssueDate"`
	Idissueplace         string `json:"idIssuePlace"`
	Nationality          string `json:"nationality"`
	Navaddress           string `json:"navAddress"`
	Curaddress           string `json:"curAddress"`
	Addisplayname        string `json:"adDisplayName"`
	Positions            []struct {
		ID             string `json:"id"`
		Code           string `json:"code"`
		Nameen         string `json:"nameEn"`
		Namevn         string `json:"nameVn"`
		CsmPosCode     string `json:"csmPosCode"`
		CsmLoginName   string `json:"csmLoginName"`
		CsmEmpCode     string `json:"csmEmpCode"`
		CsmEmpFullName string `json:"csmEmpFullName"`
		JobCode        string `json:"jobCode"`
	} `json:"positions"`
}

// Authenticator defines an interface for an authenticate-able User.
type Authenticator interface {
	Identifier() string     // Identifier returns a unique reference to this user.
	HashedPassword() string // HashedPassword returns the user's password hash.
}

// SetCache sets the Cache to use for authentication tokens.
func SetCache(c cache.Cacher) {
	authCache = c
}

// HashPassword returns a hashed version of the plain-text password provided.
func HashPassword(plainText string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plainText), bcrypt.MinCost)
	if err != nil {
		return "", err
	}

	return string(hash[:]), nil
}

func ComparePassword(HashedPassword, plainTextPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(HashedPassword), []byte(plainTextPassword))
}

// Authenticate validates an Authenticator based on it's password hash and the plain-text
// password provided.
func Authenticate(a Authenticator, plainTextPassword string) (AuthenticationTokenPair, error) {
	err := ComparePassword(a.HashedPassword(), plainTextPassword)
	if err != nil {
		return AuthenticationTokenPair{}, err
	}

	// Generate and cache a new token pair for this session
	return GenerateAndStoreTokens(a)
}
func SignOut(token string) error {
	if err := authCache.Delete(getAccessTokenCacheKey(token)); err != nil {
		return err
	}
	if err := authCache.Delete(getProfileCacheKey(token)); err != nil {
		return err
	}
	if a, err := authCache.GetString(getAccessTokenWithRefreshCacheKey(token)); err != nil {
		return err
	} else if err := authCache.Delete(getRefreshTokenCacheKey(a)); err != nil {
		return err
	}
	if err := authCache.Delete(getAccessTokenWithRefreshCacheKey(token)); err != nil {
		return err
	}
	return nil
}

// Refresh generates a new token pair for a given authenticator.
func Refresh(a Authenticator, oldAccessToken, refreshToken string) (AuthenticationTokenPair, error) {
	newTokens, err := GenerateNewTokenFromRefreshAndStoreTokens(a, refreshToken)
	if err != nil {
		return AuthenticationTokenPair{}, err
	}

	if oldAccessToken != "" {
		if err := clearCachedTokens(oldAccessToken); err != nil {
			return AuthenticationTokenPair{}, err
		}
	}

	// Clear the old tokens from the cache

	return newTokens, nil
}

func GetUserProfileCache(token string, userProfile interface{}) error {
	return authCache.GetMarshaled(getProfileCacheKey(token), userProfile)
}
func SetUserProfileCache(token string, userProfile interface{}) (interface{}, error) {
	expiresTime := time.Duration(3600)
	if getTime, err := strconv.Atoi(local.Getenv("TOKEN_EXPIRES_TIME")); err == nil {
		expiresTime = time.Duration(getTime)
	}
	save, err := authCache.PutMarshaled(getProfileCacheKey(token), userProfile)
	if err != nil {
		return nil, err
	} else if err := authCache.Expire(getProfileCacheKey(token), expiresTime); err != nil {
		return nil, err
	}
	return save, nil
}
func GetIdentifierForProfileKey(token string) (string, error) {
	return authCache.GetString(getProfileCacheKey(token))
}

// GetIdentifierForAccessToken returns a user's identifier, as returned by
// the Authenticator interface, if it exists in the cache.
//
// If the identifier does not exist, and empty string and error will be returned.
func GetIdentifierForAccessToken(a string) (string, error) {
	return authCache.GetString(getAccessTokenCacheKey(a))
}

// GetIdentifierForRefreshToken returns a user's identifier, as returned by
// the Authenticator interface, if it exists in the cahce.
//
// If the identifier does not exist, an empty string and error will be returned.
func GetIdentifierForRefreshToken(r string) (string, error) {
	return authCache.GetString(getRefreshTokenCacheKey(r))
}

func GetRefreshTokenFromAccessToken(r string) (string, error) {
	return authCache.GetString(getAccessTokenWithRefreshCacheKey(r))
}

// generateAndStoreTokens creates and caches a new AuthenticationTokenPair.
func GenerateAndStoreTokens(a Authenticator) (AuthenticationTokenPair, error) {
	t := GenerateToken()
	if err := cacheTokens(t, a); err != nil {
		return AuthenticationTokenPair{}, err
	}
	return t, nil
}

func GenerateNewTokenFromRefreshAndStoreTokens(a Authenticator, refreshToken string) (AuthenticationTokenPair, error) {
	t := GenerateToken()
	if err := cacheTokensFromRefreshToken(t, a, refreshToken); err != nil {
		return AuthenticationTokenPair{}, err
	}
	return t, nil
}

func cacheTokensFromRefreshToken(t AuthenticationTokenPair, a Authenticator, refreshToken string) error {
	expiresTime := time.Duration(3600)
	if getTime, err := strconv.Atoi(local.Getenv("TOKEN_EXPIRES_TIME")); err == nil {
		expiresTime = time.Duration(getTime)
	}
	if _, err := authCache.PutString(getAccessTokenCacheKey(t.AccessToken), a.Identifier()); err != nil {
		return err
	}
	if err := authCache.Expire(getAccessTokenCacheKey(t.AccessToken), expiresTime); err != nil {
		return err
	}
	if _, err := authCache.PutString(getAccessTokenWithRefreshCacheKey(t.AccessToken), refreshToken); err != nil {
		return err
	}
	if err := authCache.Expire(getAccessTokenWithRefreshCacheKey(t.AccessToken), expiresTime); err != nil {
		return err
	}
	return nil
}

// cacheTokens stores an access token and refresh token pair for an authenticated User.
func cacheTokens(t AuthenticationTokenPair, a Authenticator) error {
	expiresTime := time.Duration(3600)
	//logger.Log("TOKEN_EXPIRES_TIME: ", local.Getenv("TOKEN_EXPIRES_TIME"))
	if getTime, err := strconv.Atoi(local.Getenv("TOKEN_EXPIRES_TIME")); err == nil {
		expiresTime = time.Duration(getTime)
	}
	if _, err := authCache.PutString(getAccessTokenCacheKey(t.AccessToken), a.Identifier()); err != nil {
		return err
	}
	if err := authCache.Expire(getAccessTokenCacheKey(t.AccessToken), expiresTime); err != nil {
		return err
	}

	if _, err := authCache.PutString(getRefreshTokenCacheKey(t.RefreshToken), a.Identifier()); err != nil {
		return err
	}
	if err := authCache.Expire(getRefreshTokenCacheKey(t.RefreshToken), expiresTime*24); err != nil {
		return err
	}
	if _, err := authCache.PutString(getAccessTokenWithRefreshCacheKey(t.AccessToken), t.RefreshToken); err != nil {
		return err
	}
	if err := authCache.Expire(getAccessTokenWithRefreshCacheKey(t.AccessToken), expiresTime); err != nil {
		return err
	}
	//if _, err := authCache.PutString(getAccessTokenWithRefreshCacheKey(t.AccessToken), t.RefreshToken); err != nil {
	//	return err
	//}
	//if err := authCache.Expire(getAccessTokenWithRefreshCacheKey(t.AccessToken), expiresTime*2); err != nil {
	//	return err
	//}
	//if _, err := authCache.PutString(getRefreshTokenCacheKey(t.RefreshToken), a.Identifier()); err != nil {
	//	return err
	//}
	//if err := authCache.Expire(getRefreshTokenCacheKey(t.RefreshToken), expiresTime); err != nil {
	//	return err
	//}
	return nil
}

// getAccessTokenCacheKey returns the access token cache key.
func getAccessTokenCacheKey(accessToken string) string {
	return fmt.Sprintf("accessToken:%s", accessToken)
}

// getRefreshTokenCacheKey returns the refresh token cache key.
func getRefreshTokenCacheKey(refreshToken string) string {
	return fmt.Sprintf("refreshToken:%s", refreshToken)
}

// getRefreshToAccessTokenCacheKey returns the refresh -> access token cache key.
func getRefreshToAccessTokenCacheKey(refreshToken string) string {
	return fmt.Sprintf("refreshToAccessToken:%s", refreshToken)
}

func getAccessTokenWithRefreshCacheKey(accessToken string) string {
	return fmt.Sprintf("accessTokenToRefreshToken:%s", accessToken)
}

func getProfileCacheKey(token string) string {
	return fmt.Sprintf("profile:%s", token)
}

// clearCachedTokens clears all tokens associated to a refresh token.
func clearCachedTokens(r string) error {
	//logger.Log("Old Access Token: ", r)
	if t, _ := authCache.GetString(getAccessTokenWithRefreshCacheKey(r)); t != "" {
		//logger.Log("Old Access Token Info : ", t)
		if err := authCache.Delete(getAccessTokenWithRefreshCacheKey(r)); err != nil {
			return err
		}
		if err := authCache.Delete(getProfileCacheKey(r)); err != nil {
			return err
		}
		if err := authCache.Delete(getAccessTokenCacheKey(r)); err != nil {
			return err
		}
	}
	//if a, err := authCache.GetString(getAccessTokenWithRefreshCacheKey(r)); err != nil {
	//	return err
	//} else if err = authCache.Delete(getAccessTokenCacheKey(a)); err != nil {
	//	return err
	//} else if err = authCache.Delete(getRefreshTokenCacheKey(r)); err != nil {
	//	return err
	//} else if err = authCache.Delete(getRefreshToAccessTokenCacheKey(r)); err != nil {
	//	return err
	//}

	return nil
}
func GetUserIdByRequestContext(ctx context.Context) (string, error) {
	token, ok := ctx.Value(local.Getenv("JWT_TOKEN_CONTENT_KEY")).(string)
	if ok == false {
		//return "", errors.BadRequest("miss token")
	}
	userId, err := GetIdentifierForAccessToken(token)
	if err != nil || len(userId) < 12 {
		//logger.Log("GetIdentifierForAccessToken Error: ", errors.Parse(status.Convert(err).Message()).Detail)
		//return "", errors.Unauthorized("Token not correct")
	}
	return userId, nil
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
