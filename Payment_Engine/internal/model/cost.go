package model

import (
	"time"
)

// Window định nghĩa khoảng thời gian tính cost
type Window struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// UserData chứa thông tin chi phí của một người dùng hoặc hệ thống
type UserData struct {
	TotalCost      float64
	Window         Window
	NamespaceCosts map[string]float64
}

// CostData đại diện cho toàn bộ nội dung file JSON được đọc vào
type CostData map[string]interface{}

// ParseUserData xử lý dữ liệu interface{} thành UserData cụ thể
// Trả về UserData và boolean cho biết parse thành công hay không
func ParseUserData(data interface{}) (UserData, bool) {
	userDataMap, ok := data.(map[string]interface{})
	if !ok {
		return UserData{}, false
	}

	var user UserData
	user.NamespaceCosts = make(map[string]float64)
	foundTotalCost := false
	foundWindow := false

	for key, value := range userDataMap {
		switch key {
		case "totalCost":
			if cost, ok := value.(float64); ok {
				user.TotalCost = cost
				foundTotalCost = true
			}
		case "window":
			// Cẩn thận hơn khi parse window
			windowInterface, ok := value.(map[string]interface{})
			if !ok {
				continue // Bỏ qua nếu window không phải map
			}
			startStr, okS := windowInterface["start"].(string)
			endStr, okE := windowInterface["end"].(string)

			if okS && okE {
				// Thử parse với RFC3339Nano trước, sau đó RFC3339
				start, errS := time.Parse(time.RFC3339Nano, startStr)
				if errS != nil {
					start, errS = time.Parse(time.RFC3339, startStr) // Thử định dạng không có nano giây
				}

				end, errE := time.Parse(time.RFC3339Nano, endStr)
				if errE != nil {
					end, errE = time.Parse(time.RFC3339, endStr)
				}

				// Chỉ đánh dấu là tìm thấy nếu cả hai parse thành công
				if errS == nil && errE == nil {
					user.Window.Start = start
					user.Window.End = end
					foundWindow = true
				} else {
					// Ghi log hoặc xử lý lỗi nếu cần khi parse time thất bại
					// fmt.Printf("Warning: Could not parse window times for key. Start error: %v, End error: %v\n", errS, errE)
				}
			}

		default:
			// Giả định các key còn lại là namespace cost nếu là float64
			if nsCost, ok := value.(float64); ok {
				user.NamespaceCosts[key] = nsCost
			}
		}
	}

	// Chỉ trả về true nếu cả totalCost và window hợp lệ được tìm thấy
	return user, foundTotalCost && foundWindow
}
