package controller

import (
	"io"
	"net/http"
	"strings"

	"github.com/gocroot/config"
	"github.com/gocroot/helper/at"
	"github.com/gocroot/helper/atdb"
	"github.com/gocroot/helper/ghupload"
	"github.com/gocroot/helper/watoken"
	"github.com/gocroot/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func MenuUploadFileHandler(w http.ResponseWriter, r *http.Request) {
	var respn model.Response
	payload, err := watoken.Decode(config.PublicKeyWhatsAuth, at.GetLoginFromHeader(r))
	if err != nil {
		respn.Status = "Error : Token Tidak Valid"
		respn.Info = at.GetSecretFromHeader(r)
		respn.Location = "Decode Token Error"
		respn.Response = err.Error()
		at.WriteJSON(w, http.StatusForbidden, respn)
		return
	}
	_, err = atdb.GetOneDoc[model.Userdomyikado](config.Mongoconn, "user", primitive.M{"phonenumber": payload.Id})
	if err != nil {
		respn.Status = "Error : Data user tidak di temukan"
		respn.Response = err.Error()
		at.WriteJSON(w, http.StatusNotImplemented, respn)
		return
	}
	lapakid := at.GetParam(r)
	objectId, _ := primitive.ObjectIDFromHex(lapakid)
	prj, err := atdb.GetOneDoc[model.Project](config.Mongoconn, "project", primitive.M{"_id": objectId})
	if err != nil {
		respn.Status = "Error : Data lapak tidak di temukan"
		respn.Response = err.Error()
		at.WriteJSON(w, http.StatusNotImplemented, respn)
		return
	}

	file, header, err := r.FormFile("menufile")
	if err != nil {
		respn.Status = "Error : File tidak ada"
		respn.Response = err.Error()
		at.WriteJSON(w, http.StatusBadRequest, respn)
		return
	}
	defer file.Close()
	// Read the file content
	fileContent, err := io.ReadAll(file)
	if err != nil {
		respn.Status = "Error : File tidak bisa dibaca"
		respn.Response = err.Error()
		at.WriteJSON(w, http.StatusInternalServerError, respn)
		return
	}
	// Calculate hash of the file content
	hashedFileName := ghupload.CalculateHash(fileContent)

	// Get GitHub credentials and other details from the request or environment variables
	GitHubAccessToken := config.GHAccessToken
	GitHubAuthorName := "Rolly Maulana Awangga"
	GitHubAuthorEmail := "awangga@gmail.com"
	githubOrg := "gobizid"
	githubRepo := "img"
	pathFile := prj.Name + "/menu/" + hashedFileName + header.Filename[strings.LastIndex(header.Filename, "."):] // Append the original file extension
	replace := true

	// Use GithubUpload function to upload the file to GitHub
	content, _, err := ghupload.GithubUpload(GitHubAccessToken, GitHubAuthorName, GitHubAuthorEmail, fileContent, githubOrg, githubRepo, pathFile, replace)
	if err != nil {
		respn.Status = "Error : File tidak bisa diupload ke github"
		respn.Response = err.Error()
		at.WriteJSON(w, http.StatusInternalServerError, respn)
		return
	}

	// Respond with success message
	respn.Info = hashedFileName
	respn.Location = "/" + githubRepo + "/" + *content.Content.Path
	respn.Response = *content.Content.URL
	respn.Status = *content.Content.HTMLURL
	at.WriteJSON(w, http.StatusOK, respn)
}
