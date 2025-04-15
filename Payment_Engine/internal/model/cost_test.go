package model // Hoặc package model_test nếu bạn muốn test như một package bên ngoài

import (
	"reflect" // Dùng để so sánh sâu các map/struct
	"testing"
	"time"
	// Quan trọng: Đảm bảo import đúng package model của bạn nếu dùng model_test
	// "payment-engine/internal/model" // Ví dụ
)

func TestParseUserData(t *testing.T) {
	// --- Chuẩn bị các giá trị thời gian mẫu ---
	// Dùng Parse để chắc chắn định dạng đúng và có kiểu time.Time
	validStartNanoStr := "2025-04-10T05:05:45.255469311+07:00"
	validEndNanoStr := "2025-04-10T15:05:45.255469311+07:00"
	validStartRFC3339Str := "2025-04-11T08:00:00+07:00"
	validEndRFC3339Str := "2025-04-11T18:00:00+07:00"

	// Parse trước các giá trị time mong đợi
	expectedStartNano, _ := time.Parse(time.RFC3339Nano, validStartNanoStr)
	expectedEndNano, _ := time.Parse(time.RFC3339Nano, validEndNanoStr)
	expectedStartRFC3339, _ := time.Parse(time.RFC3339, validStartRFC3339Str)
	expectedEndRFC3339, _ := time.Parse(time.RFC3339, validEndRFC3339Str)

	// --- Định nghĩa các test case ---
	testCases := []struct {
		name         string      // Tên của test case
		input        interface{} // Input cho hàm ParseUserData
		wantUser     UserData    // UserData mong đợi trả về
		wantOk       bool        // Giá trị boolean mong đợi trả về
		checkWindow  bool        // Có cần kiểm tra chi tiết Window không?
		checkNsCosts bool        // Có cần kiểm tra chi tiết NamespaceCosts không?
	}{
		{
			name: "Valid full data with nano second precision",
			input: map[string]interface{}{
				"ns1-us1":   0.02329, // Có thể dùng giá trị ngắn gọn hơn
				"ns2-us1":   0.01510,
				"totalCost": 0.03839,
				"window": map[string]interface{}{
					"start": validStartNanoStr,
					"end":   validEndNanoStr,
				},
				"ignored_field": "some string", // Trường này sẽ bị bỏ qua
			},
			wantUser: UserData{
				TotalCost: 0.03839,
				Window: Window{
					Start: expectedStartNano,
					End:   expectedEndNano,
				},
				NamespaceCosts: map[string]float64{
					"ns1-us1": 0.02329,
					"ns2-us1": 0.01510,
				},
			},
			wantOk:       true,
			checkWindow:  true,
			checkNsCosts: true,
		},
		{
			name: "Valid full data with RFC3339 precision",
			input: map[string]interface{}{
				"app-prod":  1.25,
				"totalCost": 1.25,
				"window": map[string]interface{}{
					"start": validStartRFC3339Str,
					"end":   validEndRFC3339Str,
				},
			},
			wantUser: UserData{
				TotalCost: 1.25,
				Window: Window{
					Start: expectedStartRFC3339,
					End:   expectedEndRFC3339,
				},
				NamespaceCosts: map[string]float64{
					"app-prod": 1.25,
				},
			},
			wantOk:       true,
			checkWindow:  true,
			checkNsCosts: true,
		},
		{
			name: "Valid data without namespace costs",
			input: map[string]interface{}{
				"totalCost": 0.10,
				"window": map[string]interface{}{
					"start": validStartNanoStr,
					"end":   validEndNanoStr,
				},
			},
			wantUser: UserData{
				TotalCost: 0.10,
				Window: Window{
					Start: expectedStartNano,
					End:   expectedEndNano,
				},
				NamespaceCosts: make(map[string]float64), // Mong đợi map rỗng đã khởi tạo
			},
			wantOk:       true,
			checkWindow:  true,
			checkNsCosts: true,
		},
		{
			name: "Missing totalCost",
			input: map[string]interface{}{
				"ns1-us1": 0.02,
				"window": map[string]interface{}{
					"start": validStartNanoStr,
					"end":   validEndNanoStr,
				},
			},
			wantUser:     UserData{}, // Mong đợi giá trị zero
			wantOk:       false,
			checkWindow:  false,
			checkNsCosts: false,
		},
		{
			name: "Missing window",
			input: map[string]interface{}{
				"ns1-us1":   0.02,
				"totalCost": 0.02,
			},
			wantUser:     UserData{}, // Mong đợi giá trị zero
			wantOk:       false,
			checkWindow:  false,
			checkNsCosts: false,
		},
		{
			name: "Invalid type for totalCost",
			input: map[string]interface{}{
				"totalCost": "not a float", // Sai kiểu dữ liệu
				"window": map[string]interface{}{
					"start": validStartNanoStr,
					"end":   validEndNanoStr,
				},
			},
			wantUser:     UserData{},
			wantOk:       false,
			checkWindow:  false,
			checkNsCosts: false,
		},
		{
			name: "Invalid type for window",
			input: map[string]interface{}{
				"totalCost": 0.05,
				"window":    "not a map", // Sai kiểu dữ liệu
			},
			wantUser:     UserData{},
			wantOk:       false,
			checkWindow:  false,
			checkNsCosts: false,
		},
		{
			name: "Invalid type for window start",
			input: map[string]interface{}{
				"totalCost": 0.05,
				"window": map[string]interface{}{
					"start": 12345, // Sai kiểu dữ liệu
					"end":   validEndNanoStr,
				},
			},
			wantUser:     UserData{},
			wantOk:       false, // Vì window không parse được hoàn chỉnh
			checkWindow:  false,
			checkNsCosts: false,
		},
		{
			name: "Invalid format for window start",
			input: map[string]interface{}{
				"totalCost": 0.05,
				"window": map[string]interface{}{
					"start": "invalid-date-format", // Sai định dạng
					"end":   validEndNanoStr,
				},
			},
			wantUser:     UserData{},
			wantOk:       false, // Vì window không parse được hoàn chỉnh
			checkWindow:  false,
			checkNsCosts: false,
		},
		{
			name: "Missing start in window",
			input: map[string]interface{}{
				"totalCost": 0.05,
				"window": map[string]interface{}{
					// Thiếu "start"
					"end": validEndNanoStr,
				},
			},
			wantUser:     UserData{},
			wantOk:       false, // Vì window không parse được hoàn chỉnh
			checkWindow:  false,
			checkNsCosts: false,
		},
		{
			name:         "Input is not a map",
			input:        "this is a string", // Input không phải map
			wantUser:     UserData{},
			wantOk:       false,
			checkWindow:  false,
			checkNsCosts: false,
		},
		{
			name:         "Input is nil",
			input:        nil, // Input là nil
			wantUser:     UserData{},
			wantOk:       false,
			checkWindow:  false,
			checkNsCosts: false,
		},
		{
			name:         "Input is an empty map",
			input:        map[string]interface{}{}, // Input là map rỗng
			wantUser:     UserData{},
			wantOk:       false, // Vì thiếu totalCost và window
			checkWindow:  false,
			checkNsCosts: false,
		},
	}

	// --- Chạy các test case ---
	for _, tc := range testCases {
		// Sử dụng t.Run để tạo subtest cho mỗi case
		t.Run(tc.name, func(t *testing.T) {
			// Gọi hàm cần test
			gotUser, gotOk := ParseUserData(tc.input)

			// 1. Kiểm tra giá trị boolean trả về (ok)
			if gotOk != tc.wantOk {
				t.Errorf("ParseUserData() ok = %v, want %v", gotOk, tc.wantOk)
				// Nếu ok không như mong đợi, thường không cần kiểm tra user data nữa
				// nhưng vẫn có thể hữu ích để xem giá trị trả về là gì
				// t.Logf("Returned user data: %+v", gotUser)
				return // Có thể dừng subtest ở đây nếu muốn
			}

			// Chỉ kiểm tra chi tiết UserData nếu wantOk là true (và gotOk cũng là true)
			if tc.wantOk {
				// 2. Kiểm tra TotalCost
				// Lưu ý so sánh float có thể cần dung sai nhỏ trong một số trường hợp phức tạp,
				// nhưng ở đây gán trực tiếp nên so sánh bằng == là đủ.
				if gotUser.TotalCost != tc.wantUser.TotalCost {
					t.Errorf("ParseUserData() got TotalCost = %v, want %v", gotUser.TotalCost, tc.wantUser.TotalCost)
				}

				// 3. Kiểm tra Window (nếu cần)
				if tc.checkWindow {
					// Dùng Equal() để so sánh time.Time
					if !gotUser.Window.Start.Equal(tc.wantUser.Window.Start) {
						t.Errorf("ParseUserData() got Window.Start = %v, want %v", gotUser.Window.Start, tc.wantUser.Window.Start)
					}
					if !gotUser.Window.End.Equal(tc.wantUser.Window.End) {
						t.Errorf("ParseUserData() got Window.End = %v, want %v", gotUser.Window.End, tc.wantUser.Window.End)
					}
				}

				// 4. Kiểm tra NamespaceCosts (nếu cần)
				if tc.checkNsCosts {
					// Dùng reflect.DeepEqual để so sánh map
					// Đảm bảo wantUser.NamespaceCosts không phải là nil nếu mong đợi map rỗng
					expectedNsCosts := tc.wantUser.NamespaceCosts
					if expectedNsCosts == nil {
						expectedNsCosts = make(map[string]float64) // Khởi tạo nếu cần
					}
					if !reflect.DeepEqual(gotUser.NamespaceCosts, expectedNsCosts) {
						t.Errorf("ParseUserData() got NamespaceCosts = %v, want %v", gotUser.NamespaceCosts, expectedNsCosts)
					}
				}
			}
			// Nếu wantOk là false, chúng ta đã kiểm tra ở trên, không cần kiểm tra chi tiết UserData nữa
			// vì nó không có ý nghĩa.
		})
	}
}
