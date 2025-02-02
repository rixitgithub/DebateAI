# Project Setup Guide

## Backend Setup

1. Navigate to the backend directory:
   ```sh
   cd ./backend
   ```
2. Run the backend server:
   ```sh
   go run cmd/server/main.go
   ```

## Frontend Setup

1. Navigate to the frontend directory:
   ```sh
   cd ./frontend
   ```
2. Install dependencies:
   ```sh
   npm install
   ```
3. Start the development server:
   ```sh
   npm run dev
   ```

## Setting Up Amazon Cognito

Follow these steps to configure Amazon Cognito for authentication:

1. **Navigate to Cognito**
   - Go to the [AWS Management Console](https://aws.amazon.com/console/) and open Cognito.

2. **Create a User Pool**
   - Configure authentication settings as per your requirements.

3. **Retrieve Credentials**
   - Once the User Pool is set up, obtain the necessary credentials:
     - **User Pool ID**
     - **App Client ID**

4. **Update Application Configuration**
   - Add the retrieved credentials to your application's configuration file (e.g., `config.yml`).

For more details, refer to the [official AWS documentation](https://docs.aws.amazon.com/cognito/).

---

This guide follows the approach used in the project implementation. If you encounter any issues, check the AWS documentation or relevant project files.

