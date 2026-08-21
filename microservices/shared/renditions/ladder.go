package renditions

var ladder = []int{144, 240, 360, 480, 720, 1080, 1440}

// Heights returns every standard rendition no taller than the source. Tiny
// sources retain one even height so transcoding never upscales them.
func Heights(sourceHeight int) []int {
	if sourceHeight < 1 {
		return nil
	}
	if sourceHeight < ladder[0] {
		height := sourceHeight
		if height%2 != 0 {
			height--
		}
		if height < 2 {
			height = 2
		}
		return []int{height}
	}
	result := make([]int, 0, len(ladder))
	for _, height := range ladder {
		if height <= sourceHeight {
			result = append(result, height)
		}
	}
	return result
}
