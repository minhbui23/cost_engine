document.addEventListener('DOMContentLoaded', () => {
    const windowStartEl = document.getElementById('window-start');
    const windowEndEl = document.getElementById('window-end');
    const lastUpdatedEl = document.getElementById('last-updated');
    const chartCtx = document.getElementById('user-cost-chart').getContext('2d');

    function formatDate(dateStr) {
        return new Date(dateStr).toLocaleString();
    }

    function extractUserFromNamespace(ns) {
        const match = ns.match(/ns\d+-us\d+/);
        return match ? match[0].split('-')[1] : 'system'; // 'usX' là user id

    }

    function loadCostData() {

        fetch('/data/costs.json')

            .then(response => {
                if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`);
                const lastModified = response.headers.get('Last-Modified');
                lastUpdatedEl.textContent = lastModified ? formatDate(lastModified) : formatDate(new Date());
                return response.json();
            })

            .then(data => {

                if (!Array.isArray(data) || data.length === 0) {
                    throw new Error("No data available");
                }

                // Set time window (from first entry)
                const window = data[0].window || {};
                windowStartEl.textContent = window.start ? formatDate(window.start) : 'N/A';
                windowEndEl.textContent = window.end ? formatDate(window.end) : 'N/A';
                // Group costs by user > namespace

                const userData = {};
                data.forEach(entry => {
                    const ns = entry.namespace || 'unknown';
                    const user = extractUserFromNamespace(ns);
                    if (!userData[user]) userData[user] = {};
                    userData[user][ns] = (userData[user][ns] || 0) + (entry.totalCost || 0);
                });

                const users = Object.keys(userData);
                const allNamespaces = Array.from(new Set(users.flatMap(u => Object.keys(userData[u]))));

                const datasets = allNamespaces.map(ns => ({
                    label: ns,
                    data: users.map(user => userData[user][ns] || 0),
                    backgroundColor: randomRGBA(), // random color
                    stack: 'total'
                }));



                new Chart(chartCtx, {
                    type: 'bar',
                    data: {
                        labels: users,
                        datasets: datasets
                    },
                    options: {
                        responsive: true,
                        plugins: {
                            title: {
                                display: true,
                                text: 'User Cost Breakdown by Namespace (Stacked)'
                            },
                            tooltip: {
                                mode: 'index',
                                intersect: false
                            }
                        },

                        scales: {
                            x: {
                                stacked: true,
                                title: {
                                    display: true,
                                    text: 'User'
                                }
                            },

                            y: {
                                stacked: true,
                                title: {
                                    display: true,
                                    text: 'Total Cost ($)'
                                }
                            }
                        }
                    }
                });
            })

            .catch(error => {
                console.error("Error fetching or rendering cost data:", error);
                lastUpdatedEl.textContent = 'Error';
            });

    }



    function randomRGBA() {
        const r = Math.floor(Math.random() * 200);
        const g = Math.floor(Math.random() * 200);
        const b = Math.floor(Math.random() * 200);
        return `rgba(${r},${g},${b},0.6)`;
    }
    
    loadCostData();

});