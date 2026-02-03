const express = require('express');
const app = express();

app.get('/', (req, res) => {
  res.send('Hello from Jerry!');
});

app.get('/health', (req, res) => {
  res.status(200).send('ok');
});

const PORT = process.env.PORT || 3000;
app.listen(PORT, () => {
  console.log(`Server running on port ${PORT}`);
});
