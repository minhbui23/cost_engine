document.addEventListener('DOMContentLoaded', () => {
    
    const windowStartEl = document.getElementById('window-start');
    const windowEndEl = document.getElementById('window-end');
    const lastUpdatedEl = document.getElementById('last-updated');
    const chartCanvas = document.getElementById('user-cost-chart');
    const windowSelect = document.getElementById('window-select');
    const refreshButton = document.getElementById('refresh-button');

    let currentChart = null; 
    const API_BASE_URL = '/getcost'; 

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
        if (!selectedWindow) {
            alert("Please select a time window.");
            return;
        }

        windowStartEl.textContent = 'Loading...';
        windowEndEl.textContent = 'Loading...';
        lastUpdatedEl.textContent = new Date().toLocaleString();

        if (currentChart) {
            currentChart.destroy();
            currentChart = null;
        }

        const apiUrl = `${API_BASE_URL}?window=${selectedWindow}`; 

        console.log(`Workspaceing data from: ${apiUrl}`); 

        fetch(apiUrl)
            .then(response => {
                if (!response.ok) {
                    return response.text().then(text => {
                         throw new Error(`HTTP error ${response.status}: ${text || response.statusText}`);
                    });
                }
                return response.json();
            })
            .then(data => {
                console.log("Data received:", data); 

                if (!data || Object.keys(data).length === 0) {
                    windowStartEl.textContent = 'N/A';
                    windowEndEl.textContent = 'N/A';
                    const ctx = chartCanvas.getContext('2d');
                    ctx.clearRect(0, 0, chartCanvas.width, chartCanvas.height); 
                    ctx.font = '16px Arial';
                    ctx.fillStyle = '#666';
                    ctx.textAlign = 'center';
                    ctx.fillText('No data available for the selected window.', chartCanvas.width / 2, chartCanvas.height / 2);
                    console.warn("No data received from API for the selected window.");
                    return; 
                }


                const users = Object.keys(data); 

                const firstUserKey = users[0];
                const windowInfo = data[firstUserKey]?.window || {};
                windowStartEl.textContent = formatDate(windowInfo.start);
                windowEndEl.textContent = formatDate(windowInfo.end);

                const allNamespaces = new Set();
                users.forEach(user => {
                    Object.keys(data[user]).forEach(key => {
                        if (key !== 'totalCost' && key !== 'window') {
                            allNamespaces.add(key);
                        }
                    });
                });
                const uniqueNamespaces = Array.from(allNamespaces);
                console.log("Unique Namespaces:", uniqueNamespaces);


                const datasets = uniqueNamespaces.map(ns => {
                    const namespaceData = users.map(user => {
                        return data[user]?.[ns] || 0;
                    });
                    console.log(`Dataset for ${ns}:`, namespaceData);
                    return {
                        label: ns, 
                        data: namespaceData,
                        backgroundColor: randomRGBA(),
                        stack: 'userStack' 
                    };
                });

                const ctx = chartCanvas.getContext('2d');
                currentChart = new Chart(ctx, {
                    type: 'bar',
                    data: {
                        labels: users, 
                        datasets: datasets 
                    },
                    options: {
                        responsive: true,
                        maintainAspectRatio: false, 
                        plugins: {
                            title: {
                                display: true,
                                text: `User Cost Breakdown by Namespace (Window: ${selectedWindow})`
                            },
                            tooltip: {
                                mode: 'index', 
                                intersect: false
                            },
                            legend: {
                                display: true, 
                                position: 'top',
                            }
                        },
                        scales: {
                            x: {
                                stacked: true, 
                                title: {
                                    display: true,
                                    text: 'User Group'
                                }
                            },
                            y: {
                                stacked: true, 
                                title: {
                                    display: true,
                                    text: 'Total Cost ($)'
                                },
                                beginAtZero: true 
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
                const ctx = chartCanvas.getContext('2d');
                ctx.clearRect(0, 0, chartCanvas.width, chartCanvas.height); 
                ctx.font = '16px Arial';
                ctx.fillStyle = 'red';
                ctx.textAlign = 'center';
                ctx.fillText(`Error loading data: ${error.message}`, chartCanvas.width / 2, chartCanvas.height / 2, chartCanvas.width - 20); // Wrap text slightly
            });
    }

    refreshButton.addEventListener('click', loadCostData);

    loadCostData();
});