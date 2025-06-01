<p align="left">
  <img src="https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcRpPlI1P7SK3pemg67VMPbvNzxYyk0UtlmJiQ&s" alt="Aossie Logo" height="120"/>
</p>

<h1 align="right">
  <b>AOSSIE | DebateAI</b>
</h1>

---

##  About DebateAI

**DebateAI** is an AI-enhanced, real-time debating platform designed to sharpen communication skills. Whether competing against human opponents or LLM-powered AI challengers, users can participate in structured debates that mimic formal competitions.

###  Key Features

-  **User vs. User Debates**
  - Real-time debates via **WebSockets**, **WebRTC** (audio/video/text)
  - Structured formats: **opening**, **cross-exam**, and **closing**
  
-  **User vs. AI Debates**
  - LLM-generated counterarguments that adapt to your input

-  **Custom Debate Rooms**
  - Create private, topic-specific debate spaces

---

##  Project Setup Guide

###  Backend Configuration

1. Create a file named `config.prod.yml` in the `backend/config/` directory.

Paste the following configuration:
```
server:
  port: 1313  # The port number your backend server will run on

database:
  uri: "mongodb+srv://<username>:<password>@<cluster-url>/<database-name>"  
  # Replace with your MongoDB Atlas connection string
  # Get this from your MongoDB Atlas dashboard after creating a cluster and database

gemini:
  apiKey: "<YOUR_GEMINI_API_KEY>"
  # API key for OpenAI / Gemini model access
  # Obtain from your OpenRouter.ai or OpenAI account dashboard

jwt:
  secret: "<YOUR_JWT_SECRET>"
  # A secret string used to sign JWT tokens
  # Generate a strong random string (e.g. use `openssl rand -hex 32`)

  expiry: 1440  
  # Token expiry time in minutes (e.g. 1440 = 24 hours)

smtp:
  host: "smtp.gmail.com"  
  # SMTP server host for sending emails (example is Gmail SMTP)

  port: 587  
  # SMTP server port (587 for TLS)

  username: "<YOUR_EMAIL_ADDRESS>"  
  # Email username (your email address)

  password: "<YOUR_EMAIL_PASSWORD_OR_APP_PASSWORD>"  
  # Password for the email or app-specific password if 2FA is enabled

  senderEmail: "<YOUR_EMAIL_ADDRESS>"  
  # The 'from' email address used when sending mails

  senderName: "DebateAI Team"  

googleOAuth:
  clientID: "<YOUR_GOOGLE_OAUTH_CLIENT_ID>"  
  # Google OAuth Client ID for OAuth login
  # Obtain from Google Cloud Console (APIs & Services > Credentials > OAuth 2.0 Client IDs)
```

>  **Note**: Do **not** commit this file to a public repository. Use `.gitignore`.

---

###  Running the Backend (Go)

1. Navigate to the backend folder:
   ```
   cd backend
   ```

2. Initialize Go modules (if not already done):
   ```
   go mod tidy
   ```

3. Run the server:
   ```
   go run cmd/server/main.go
   ```

---

###  Running the Frontend (React + Vite)

1. Open a new terminal and navigate to the frontend directory:
   ```
   cd frontend
   ```

2. Install dependencies:
   ```
   npm install
   ```

3. Create a `.env` file and add:
   ```
   VITE_BASE_URL="http://localhost:1313"
   ```

4. Start the development server:
   ```
   npm run dev
   ```

---

##  Contributing

Want to contribute to DebateAI? Reach out to [Bhavik Mangla](https://github.com/bhavik-mangla) and [Rishit Tiwari](https://github.com/rixitgithub). Contributions that enhance accessibility, features, or performance are always welcome.

---

##  License

MIT © [AOSSIE](https://aossie.org)

---
