package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"thugcorp.io/grocery/api/internal/respond"
)

type presignReq struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
}

type presignResp struct {
	UploadURL string `json:"upload_url"`
	PublicURL string `json:"public_url"`
}

// POST /v1/uploads/presign
func (h *Handlers) PresignUpload(w http.ResponseWriter, r *http.Request) {

	var body presignReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.FileName == "" {
		respond.Error(w, http.StatusBadRequest, "file_name is required")
		return
	}
	if body.ContentType == "" {
		body.ContentType = "application/octet-stream"
	}

	accountID := os.Getenv("R2_ACCOUNT_ID")
	bucket    := os.Getenv("R2_BUCKET")
	keyID     := os.Getenv("R2_ACCESS_KEY_ID")
	secret    := os.Getenv("R2_SECRET_ACCESS_KEY")
	pubBase   := os.Getenv("R2_PUBLIC_URL")

	if accountID == "" || bucket == "" || keyID == "" || secret == "" {
		respond.Error(w, http.StatusServiceUnavailable, "image storage not configured")
		return
	}

	ext := ""
	if idx := strings.LastIndex(body.FileName, "."); idx >= 0 {
		ext = body.FileName[idx:]
	}
	objectKey := "products/" + randomHex() + ext

	uploadURL := r2PresignPUT(accountID, bucket, objectKey, keyID, secret, 3600)
	publicURL := strings.TrimRight(pubBase, "/") + "/" + objectKey

	respond.JSON(w, http.StatusOK, presignResp{UploadURL: uploadURL, PublicURL: publicURL})
}

func r2PresignPUT(accountID, bucket, objectKey, keyID, secret string, ttl int) string {
	now      := time.Now().UTC()
	datestamp := now.Format("20060102")
	amzDate   := now.Format("20060102T150405Z")
	region   := "auto"
	service  := "s3"
	host     := accountID + ".r2.cloudflarestorage.com"

	credentialScope := datestamp + "/" + region + "/" + service + "/aws4_request"
	credential      := keyID + "/" + credentialScope

	canonicalURI := "/" + bucket + "/" + strings.TrimPrefix(objectKey, "/")
	signedHeaders := "host"

	// Canonical query string — params must be alphabetically sorted
	canonicalQS := strings.Join([]string{
		"X-Amz-Algorithm=AWS4-HMAC-SHA256",
		"X-Amz-Credential=" + awsEncode(credential),
		"X-Amz-Date=" + amzDate,
		"X-Amz-Expires=" + fmt.Sprintf("%d", ttl),
		"X-Amz-SignedHeaders=" + signedHeaders,
	}, "&")

	canonicalRequest := strings.Join([]string{
		"PUT",
		canonicalURI,
		canonicalQS,
		"host:" + host + "\n",
		signedHeaders,
		"UNSIGNED-PAYLOAD",
	}, "\n")

	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + credentialScope + "\n" + hexSHA256([]byte(canonicalRequest))

	sigKey := awsDeriveKey(secret, datestamp, region, service)
	sig    := hex.EncodeToString(awsHMAC(sigKey, []byte(stringToSign)))

	return "https://" + host + canonicalURI + "?" + canonicalQS + "&X-Amz-Signature=" + sig
}

// awsEncode percent-encodes a string for use in AWS canonical query strings.
// All chars except A-Z a-z 0-9 - _ . ~ are encoded (including /).
func awsEncode(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteByte(c)
		} else {
			b.WriteString(fmt.Sprintf("%%%02X", c))
		}
	}
	return b.String()
}

func hexSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func awsHMAC(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

func awsDeriveKey(secret, date, region, service string) []byte {
	kDate    := awsHMAC([]byte("AWS4"+secret), []byte(date))
	kRegion  := awsHMAC(kDate, []byte(region))
	kService := awsHMAC(kRegion, []byte(service))
	return awsHMAC(kService, []byte("aws4_request"))
}

func randomHex() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
