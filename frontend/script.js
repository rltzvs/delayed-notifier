document.getElementById('notify-form').addEventListener('submit', async function(event) {
    event.preventDefault();

    const form = event.target;
    const formData = new FormData(form);
    
    const email = formData.get('email');
    const message = formData.get('message');
    const sendAtValue = formData.get('send_at');

    const sendAt = new Date(sendAtValue).toISOString();

    const data = {
        email: email,
        message: message,
        send_at: sendAt
    };

    const responseDiv = document.getElementById('response');
    responseDiv.className = 'response';
    responseDiv.textContent = '';

    try {
        const response = await fetch(`${window.APP_CONFIG.API_URL}/notify`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(data)
        });

        if (response.ok) {
            const result = await response.json();
            responseDiv.className = 'response success';
            responseDiv.textContent = `Уведомление успешно создано! ID: ${result.id}`;
            form.reset();
        } else {
            const error = await response.json();
            responseDiv.className = 'response error';
            responseDiv.textContent = `Ошибка: ${error.message || response.statusText}`;
        }
    } catch (error) {
        responseDiv.className = 'response error';
        responseDiv.textContent = `Произошла ошибка сети: ${error.message}`;
    }
}); 