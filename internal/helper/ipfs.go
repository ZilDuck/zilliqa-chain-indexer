package helper

import (
	"net/url"
	"regexp"
)

func IsUrl(uri string) bool {
	u, err := url.Parse(uri)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func IsIpfs(uri string) bool {
	re := regexp.MustCompile("(Qm[1-9A-HJ-NP-Za-km-z]{44}.*$)")
	parts := re.FindStringSubmatch(uri)
	if len(parts) == 2 {
		return true
	}

	if !IsUrl(uri) {
		return false
	}

	u, _ := url.Parse(uri)
	if u.Scheme == "ipfs" {
		return true
	}

	return false
}

func GetIpfs(ipfsUri string) *string {
	re := regexp.MustCompile("(Qm[1-9A-HJ-NP-Za-km-z]{44}.*$)")
	parts := re.FindStringSubmatch(ipfsUri)
	if len(parts) == 2 {
		ipfsUri = "ipfs://" + parts[1]
		return &ipfsUri
	}

	if len(ipfsUri) >=7 && ipfsUri[:7] == "ipfs://" {
		return &ipfsUri
	}

	return nil
}
