package stopwords

import "github.com/adnilis/x-hmsw/utils/stopwords/data"

// IsStopWord 检查给定的单词是否是指定语言的停用词
func IsStopWord(lang, word string) bool {
	_, ok := data.Languages[lang][word]

	return ok
}
