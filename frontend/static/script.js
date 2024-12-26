document.getElementById('command').addEventListener('keypress', function(e) {
    if (e.key === 'Enter') {
        const command = e.target.value;
        if (command) {
            sendCommand(command);
            e.target.value = ''; // Clear the input field
        }
    }
});

function sendCommand(command) {
    const output = document.getElementById('output');

    // Display the entered command
    output.innerHTML += `> ${command}\n`;
    output.scrollTop = output.scrollHeight; // Scroll to the bottom

    fetch('/execute', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ command: command }),
    })
        .then(response => {
            if (!response.ok) {
                return response.json().then(errorData => {
                    if (errorData && errorData.error) {
                        throw new Error(errorData.error);
                    }
                    throw new Error("An unknown error occurred.");
                });
            }
            return response.json();
        })
        .then(data => {
            // Display the response from the server
            output.innerHTML += `${data.response}\n`;
            output.scrollTop = output.scrollHeight; // Scroll to the bottom
        })
        .catch(error => {
            // Display any error that occurs
            output.innerHTML += `Error: ${error.message}\n`;
            output.scrollTop = output.scrollHeight;
        });
}
