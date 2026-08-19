const routeForm = document.getElementById('route-form');
const routeOutput = document.getElementById('route-output');
routeForm.addEventListener('submit', async (event) => {
  event.preventDefault();
  const form = new FormData(routeForm);
  const payload = {
    datasetId: form.get('datasetId'),
    from: form.get('from'),
    to: form.get('to'),
    mode: form.get('mode'),
    departure: form.get('departure') ? new Date(form.get('departure')).toISOString() : new Date().toISOString(),
  };
  try {
    const response = await fetch('/api/v1/route', {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(payload)});
    routeOutput.textContent = JSON.stringify(await response.json(), null, 2);
  } catch (error) {
    routeOutput.textContent = String(error);
  }
});
document.getElementById('refresh-datasets').addEventListener('click', async () => {
  const response = await fetch('/api/v1/datasets');
  document.getElementById('dataset-output').textContent = JSON.stringify(await response.json(), null, 2);
});
