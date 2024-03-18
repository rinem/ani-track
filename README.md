<div align="center">
  <h1>🚀 AniTrack - A MyAnimeList CLI Client</h1>
</div>

A command line client built with Go that interfaces MyAnimeList API for viewing top anime, anime details, updating user's anime lists and more

---

# 📌 Prerequisites

- **MyAnimeList Account** is required to obtain `ClientId` and `ClientSecret`.

---

# 🔧 Configuration

To get started, you'll need a 'Client ID' and 'Client Secret' from MyAnimeList's API Dashboard:

1. 🔗 **Navigate to**: [MyAnimeList API Dashboard](https://myanimelist.net/apiconfig)

2. 🚪 **Sign in or create a MyAnimeList account**.

3. ➕ **Click on 'Create  ID'**

4. 📜 **Fill in the app details like example below**.

![MAL API Client Example](https://raw.githubusercontent.com/rinem/ani-track/main/assets/mal-client.png)

5. 🌐 **In the App Redirect URL field of the app you created, please enter the following URLs:**

   - 📎 `http://localhost:9999/oauth/callback`

6. 🛍 **Once the App is created**, you'll find the 'Client ID' and 'Client Secret' on the app details page.

7. 🔑 **Add Credentials in .env**:
    - Create a .env file in the root directory of this project and copy these values by adding your MAL client id and client secret.
```
# MAL
MAL_CLIENT_ID="<CLIENT ID GOES HERE>"
MAL_CLIENT_SECRET="<CLIENT SECRET GOES HERE>"
REDIRECT_URL="http://localhost:9999/oauth/callback"
```

🚫 **Remember**: Keep your 'Client Secret and Client Id' confidential. Never share it! They can be used to control your MyAnimeList data.

---

# 📝 TODO List
- [x] Setup oauth with MyAnimeList API
- [ ] Use access token to authenticate User's requests and refresh token when access token expired
- [ ] Add methods for calling different API endpoints of MAL
- [ ] Integrate Cobra and add CLI commands to use different methods
- [ ] Improve UI of the CLI results

---

# 💻 Local Development

1. Install Go version 1.21 or above https://go.dev/
2. Clone repo
3. Run `go mod tidy`
4. Follow `Configuration` steps mentioned above
5. Now oauth with MyAnimeList can be executed from root directory e.g. `go run main.go`

---

# 🤝 Contributing

Your contributions are welcome! 🌟 Feel free to submit pull requests or raise issues.

---

# 📜 License

This project is under the MIT License. Dive into the `LICENSE` file for more.

