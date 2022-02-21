package helper

import "regexp"

func GetIpfs(metadataUri string) string {
	if len(metadataUri)<7 {
		return ""
	}

	if metadataUri[:7] == "ipfs://" {
		return metadataUri
	}

	re := regexp.MustCompile("(Qm[1-9A-HJ-NP-Za-km-z]{44}.*$)")
	parts := re.FindStringSubmatch(metadataUri)
	if len(parts) == 2 {
		return "ipfs://" + parts[1]
	}

	return ""
}

func IsIpfs(uri string) bool {
	if len(uri)<7 {
		return false
	}

	if uri[:7] == "ipfs://" {
		return true
	}

	re := regexp.MustCompile("(Qm[1-9A-HJ-NP-Za-km-z]{44}.*$)")
	parts := re.FindStringSubmatch(uri)
	if len(parts) == 2 {
		return true
	}

	return false
}
