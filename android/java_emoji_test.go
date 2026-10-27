package android

import (
	"slices"
	"strings"
	"testing"

	"github.com/akrck02/go-renderer/android/java"
)

// The class is loaded by its name, and compiled only if the tool is given it.
func TestTheEmojiPictureClassIsWhereItsNameSays(test *testing.T) {
	if !strings.Contains(java.EmojiPicture, "package io.github.akrck02.gorenderer;") ||
		!strings.Contains(java.EmojiPicture, "public final class EmojiPicture") ||
		!strings.HasSuffix(java.EmojiPictureClass, ".EmojiPicture") ||
		!strings.HasPrefix(java.EmojiPicturePath, strings.ReplaceAll(java.EmojiPictureClass, ".", "/")) {
		test.Fatal("the class's package, name and path disagree")
	}
	if !slices.Contains(java.Sources, java.Source{Path: java.EmojiPicturePath, Text: java.EmojiPicture}) {
		test.Fatal("the tool is not given the class to compile")
	}
}
