document.addEventListener('DOMContentLoaded', () => {
    const windowStartEl = document.getElementById('window-start');
    const windowEndEl = document.getElementById('window-end');
    const lastUpdatedEl = document.getElementById('last-updated');
    const chartCanvas = document.getElementById('user-cost-chart');
    const windowSelect = document.getElementById('window-select');
    // const stepSelect = document.getElementById('step-select'); // Nếu dùng step
    const refreshButton = document.getElementById('refresh-button');

    let currentChart = null; // Biến để lưu trữ biểu đồ hiện tại

    // --- !!! Quan trọng: Cấu hình URL của API Server ---
    // Thay đổi URL này thành địa chỉ thực tế của API container khi deploy
    // Ví dụ: 'http://cost-engine-api.yourdomain.com/getcost' hoặc 'http://localhost:9991/getcost' nếu chạy local
    const API_BASE_URL = 'http://localhost:9991/getcost'; // Dùng đường dẫn tương đối nếu UI và API cùng domain/proxy, hoặc URL tuyệt đối nếu khác
    // Ví dụ URL tuyệt đối: const API_BASE_URL = 'http://<api-server-ip-or-dns>:9991/getcost';

    function formatDate(dateStr) {
        if (!dateStr) return 'N/A';
        try {
            // Thêm xử lý nếu dateStr không hợp lệ
            const date = new Date(dateStr);
            return isNaN(date) ? 'Invalid Date' : date.toLocaleString();
        } catch (e) {
            return 'Invalid Date';
        }
    }

    function randomRGBA() {
        const r = Math.floor(Math.random() * 200);
        const g = Math.floor(Math.random() * 200);
        const b = Math.floor(Math.random() * 200);
        return `rgba(${r},${g},${b},0.6)`;
    }

    function loadCostData() {
        const selectedWindow = windowSelect.value;
        // const selectedStep = stepSelect.value; // Nếu dùng step
        if (!selectedWindow) {
            alert("Please select a time window.");
            return;
        }

        // Hiển thị trạng thái đang tải
        windowStartEl.textContent = 'Loading...';
        windowEndEl.textContent = 'Loading...';
        lastUpdatedEl.textContent = new Date().toLocaleString(); // Thời điểm bắt đầu fetch

        // Xóa biểu đồ cũ nếu có
        if (currentChart) {
            currentChart.destroy();
            currentChart = null;
        }

        // Xây dựng URL API
        const apiUrl = `${API_BASE_URL}?window=${selectedWindow}`; //&step=${selectedStep}`; // Nếu dùng step

        console.log(`Workspaceing data from: ${apiUrl}`); // Log để debug

        fetch(apiUrl)
            .then(response => {
                if (!response.ok) {
                    // Cố gắng đọc lỗi từ server nếu có
                    return response.text().then(text => {
                         throw new Error(`HTTP error ${response.status}: ${text || response.statusText}`);
                    });
                }
                return response.json();
            })
            .then(data => {
                console.log("Data received:", data); // Log dữ liệu nhận được

                if (!data || Object.keys(data).length === 0) {
                    windowStartEl.textContent = 'N/A';
                    windowEndEl.textContent = 'N/A';
                    // Hiển thị thông báo không có dữ liệu trên chart area
                    const ctx = chartCanvas.getContext('2d');
                    ctx.clearRect(0, 0, chartCanvas.width, chartCanvas.height); // Xóa canvas
                    ctx.font = '16px Arial';
                    ctx.fillStyle = '#666';
                    ctx.textAlign = 'center';
                    ctx.fillText('No data available for the selected window.', chartCanvas.width / 2, chartCanvas.height / 2);
                    console.warn("No data received from API for the selected window.");
                    return; // Dừng xử lý nếu không có dữ liệu
                }


                const users = Object.keys(data); // Lấy danh sách user groups (system, user1, user2)

                // Lấy window từ entry đầu tiên (giả định tất cả giống nhau)
                const firstUserKey = users[0];
                const windowInfo = data[firstUserKey]?.window || {};
                windowStartEl.textContent = formatDate(windowInfo.start);
                windowEndEl.textContent = formatDate(windowInfo.end);

                // --- Xử lý dữ liệu cho biểu đồ ---
                // 1. Tìm tất cả các namespace duy nhất từ tất cả user groups
                const allNamespaces = new Set();
                users.forEach(user => {
                    Object.keys(data[user]).forEach(key => {
                        // Bỏ qua các key đặc biệt như 'totalCost', 'window'
                        if (key !== 'totalCost' && key !== 'window') {
                            allNamespaces.add(key);
                        }
                    });
                });
                const uniqueNamespaces = Array.from(allNamespaces);
                console.log("Unique Namespaces:", uniqueNamespaces);

                // 2. Tạo datasets cho Chart.js
                const datasets = uniqueNamespaces.map(ns => {
                    const namespaceData = users.map(user => {
                        // Lấy chi phí của namespace 'ns' cho user 'user'
                        // Nếu user đó không có namespace này thì trả về 0
                        return data[user]?.[ns] || 0;
                    });
                    console.log(`Dataset for ${ns}:`, namespaceData);
                    return {
                        label: ns, // Tên của stack
                        data: namespaceData,
                        backgroundColor: randomRGBA(),
                        stack: 'userStack' // Đảm bảo các namespace stack trên cùng một user bar
                    };
                });

                // --- Vẽ biểu đồ ---
                const ctx = chartCanvas.getContext('2d');
                currentChart = new Chart(ctx, {
                    type: 'bar',
                    data: {
                        labels: users, // Nhãn trục X là user groups
                        datasets: datasets // Dữ liệu các stack namespace
                    },
                    options: {
                        responsive: true,
                        maintainAspectRatio: false, // Cho phép chart co giãn tốt hơn
                        plugins: {
                            title: {
                                display: true,
                                text: `User Cost Breakdown by Namespace (Window: ${selectedWindow})`
                            },
                            tooltip: {
                                mode: 'index', // Hiển thị tooltip cho cả stack khi hover vào user
                                intersect: false
                            },
                            legend: {
                                display: true, // Hiển thị chú giải cho các namespace
                                position: 'top',
                            }
                        },
                        scales: {
                            x: {
                                stacked: true, // Quan trọng: bật chế độ chồng cột cho trục X
                                title: {
                                    display: true,
                                    text: 'User Group'
                                }
                            },
                            y: {
                                stacked: true, // Quan trọng: bật chế độ chồng cột cho trục Y
                                title: {
                                    display: true,
                                    text: 'Total Cost ($)'
                                },
                                beginAtZero: true // Bắt đầu trục Y từ 0
                            }
                        }
                    }
                });
            })
            .catch(error => {
                console.error("Error fetching or rendering cost data:", error);
                windowStartEl.textContent = 'Error';
                windowEndEl.textContent = 'Error';
                lastUpdatedEl.textContent = 'Error';
                 // Hiển thị lỗi trên chart area
                const ctx = chartCanvas.getContext('2d');
                ctx.clearRect(0, 0, chartCanvas.width, chartCanvas.height); // Xóa canvas
                ctx.font = '16px Arial';
                ctx.fillStyle = 'red';
                ctx.textAlign = 'center';
                ctx.fillText(`Error loading data: ${error.message}`, chartCanvas.width / 2, chartCanvas.height / 2, chartCanvas.width - 20); // Wrap text slightly
            });
    }

    // Gắn sự kiện cho nút Refresh
    refreshButton.addEventListener('click', loadCostData);

    // Tải dữ liệu lần đầu khi trang được load
    loadCostData();
});