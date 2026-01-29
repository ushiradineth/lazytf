package components

func adjustScrollOffset(currentOffset, selectedIndex, itemsLen, itemsHeight, lastMove, anchorTop, anchorBottom int) int {
	if itemsHeight <= 0 || itemsLen == 0 {
		return 0
	}

	maxOffset := max(0, itemsLen-itemsHeight)

	switch {
	case lastMove > 0:
		threshold := currentOffset + anchorBottom
		if selectedIndex > threshold {
			currentOffset = selectedIndex - anchorBottom
		}
	case lastMove < 0:
		threshold := currentOffset + anchorTop
		if selectedIndex < threshold {
			currentOffset = selectedIndex - anchorTop
		}
	default:
		if selectedIndex < currentOffset {
			currentOffset = selectedIndex
		} else if selectedIndex >= currentOffset+itemsHeight {
			currentOffset = selectedIndex - itemsHeight + 1
		}
	}

	if currentOffset < 0 {
		return 0
	}
	if currentOffset > maxOffset {
		return maxOffset
	}
	return currentOffset
}
