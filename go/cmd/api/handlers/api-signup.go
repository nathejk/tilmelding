package handlers

import (
	"github.com/go-redis/redis"

	"nathejk.dk/cmd/api/command"
	"nathejk.dk/pkg/notification"
	"nathejk.dk/pkg/streaminterface"
)

type apiController struct {
	publisher   streaminterface.Publisher
	command     command.AllowedCommands
	redisclient *redis.Client
	sms         notification.SmsSender
	clientID    string
}

func NewApiController(publisher streaminterface.Publisher, cmd command.AllowedCommands, redisclient *redis.Client, sms notification.SmsSender) *apiController {
	c := apiController{
		publisher:   publisher,
		command:     cmd,
		redisclient: redisclient,
		sms:         sms,
		clientID:    "tilmelding-api",
	}
	return &c
}

/*
type SignupRequest struct {
	TeamType string            `json:"type"`
	Name     string            `json:"name"`
	Phone    types.PhoneNumber `json:"phone"`
	Email    types.Email       `json:"email"`
}
type SignupResponse struct {
	TeamID types.TeamID `json:"teamId"`
}

func (c *apiController) ApiSignupHandler(w http.ResponseWriter, r *http.Request) {
	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	body := messages.NathejkTeamSignedUp{
		TeamID: types.TeamID(uuid.New().String()),
		Type:   req.TeamType,
		//		Slug:    c.command.NextTeamSlug(),
		Name:    req.Name,
		Phone:   req.Phone,
		Email:   req.Email,
		Pincode: fmt.Sprintf("%d", rand.Intn(9999)+10000)[1:],
	}
	msg := c.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:signedup"))
	msg.SetBody(body)
	meta := messages.Metadata{Producer: c.clientID}
	meta.RequestHeaders.Set(r.Header)
	msg.SetMeta(&meta)
	if err := c.publisher.Publish(msg); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	go func() {
		text := "Din pincode er:" + body.Pincode
		errorText := ""
		if err := c.sms.Send(body.Phone.Normalize(), text); err != nil {
			errorText = err.Error()
		}
		_ = errorText
		msg := c.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:sms.sent"))
		msg.SetBody(messages.NathejkSmsSent{
			Phone:  types.PhoneNumber(body.Phone.Normalize()),
			TeamID: body.TeamID,
			Text:   text,
			//			Error:  errorText,
		})
		msg.SetMeta(&messages.Metadata{Producer: c.clientID})
		c.publisher.Publish(msg)
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(SignupResponse{
		TeamID: body.TeamID,
	})
}
type SignupConfirmRequest struct {
	TeamID  types.TeamID
	Phone   types.PhoneNumber
	Pincode string
}
type SignupConfirmResponse struct {
	OK bool
}

func (c *apiController) ApiSignupConfirmedHandler(w http.ResponseWriter, r *http.Request) {
	var req SignupConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	pincode, err := c.redisclient.HGet("pincodes", string(req.TeamID)).Result()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	var code struct {
		Pincode string `json:"pincode"`
	}
	json.Unmarshal([]byte(pincode), &code)
	if code.Pincode != req.Pincode {
		http.Error(w, "unauthorized", http.StatusForbidden)
		return
	}
	msg := c.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:phonenumber.confirmed"))
	msg.SetBody(messages.NathejkPhoneNumberConfirmed{
		TeamID: req.TeamID,
		Phone:  req.Phone,
	})
	meta := messages.Metadata{Producer: c.clientID}
	meta.RequestHeaders.Set(r.Header)
	msg.SetMeta(&meta)
	err = c.publisher.Publish(msg)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SignupConfirmResponse{
		OK: err == nil,
	})
}

/*
type TeamUpdateRequest struct {
	TeamID            types.TeamID      `json:"teamId"`
	Name              string            `json:"name"`
	GroupName         string            `json:"groupName"`
	Korps             string            `json:"korps"`
	AdvspejdNumber    string            `json:"advspejdNumber"`
	ContactName       string            `json:"contactName"`
	ContactAddress    string            `json:"contactAddress"`
	ContactPostalCode string            `json:"contactPostalCode"`
	ContactMail       types.Email       `json:"contactMail"`
	ContactPhone      types.PhoneNumber `json:"contactPhone"`
	ContactRole       string            `json:"contactRole"`
	Members           []struct {
		MemberID    types.MemberID    `json:"memberId"`
		Name        string            `json:"Name"`
		Address     string            `json:"address"`
		PostalCode  string            `json:"postalCode"`
		Mail        types.Email       `json:"mail"`
		Phone       types.PhoneNumber `json:"phone"`
		PhoneParent types.PhoneNumber `json:"phoneParent"`
		Birthday    types.Date        `json:"birthday"`
		Returning   bool              `json:"returning"`
	} `json:"members"`
}
type TeamUpdateResponse struct {
	OK bool
}

type TeamView struct {
	messages.NathejkTeamUpdated
	Members []messages.NathejkMemberUpdated `json:"members"`
}

func (c *apiController) ApiPatruljeCreateHandler(w http.ResponseWriter, r *http.Request) {
	//team := c.command.PatruljeCreate()
	response := TeamReadResponse{
		TeamID: types.TeamID(c.command.NextTeamSlug()),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
func (c *apiController) ApiPatruljeReadUpdateHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("METHOD %q", r.Method)
	switch r.Method {
	case http.MethodPost:
		c.ApiTeamUpdateHandler(w, r)
	case http.MethodGet:
		c.ApiTeamReadHandler(w, r)
	}
}

func (c *apiController) ApiTeamUpdateHandler(w http.ResponseWriter, r *http.Request) {
	var req TeamUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// validate teamId against existing teamIds
	team, err := c.redisclient.HGet("teams", string(req.TeamID)).Result()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	var view TeamView
	json.Unmarshal([]byte(team), &view)
	body := messages.NathejkTeamUpdated{
		TeamID:            req.TeamID,
		Type:              view.Type,
		Name:              req.Name,
		GroupName:         req.GroupName,
		Korps:             req.Korps,
		AdvspejdNumber:    req.AdvspejdNumber,
		ContactName:       req.ContactName,
		ContactAddress:    req.ContactAddress,
		ContactPostalCode: req.ContactPostalCode,
		ContactEmail:      req.ContactMail,
		ContactPhone:      req.ContactPhone,
		ContactRole:       req.ContactRole,
	}
	msg := c.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:team.updated"))
	msg.SetBody(body)
	meta := messages.Metadata{Producer: c.clientID}
	meta.RequestHeaders.Set(r.Header)
	msg.SetMeta(&meta)
	err = c.publisher.Publish(msg)

	memberIDs := map[types.MemberID]bool{}
	for _, member := range view.Members {
		memberIDs[member.MemberID] = true
	}
	for _, member := range req.Members {
		if member.MemberID != "" && !memberIDs[member.MemberID] {
			// Not allowed to edit a member not belonging to this team
			continue
		}
		if member.MemberID == "" {
			member.MemberID = types.MemberID(uuid.New().String())
		}
		delete(memberIDs, member.MemberID)
		body := messages.NathejkMemberUpdated{
			MemberID:    member.MemberID,
			TeamID:      req.TeamID,
			Name:        member.Name,
			Address:     member.Address,
			PostalCode:  member.PostalCode,
			Email:       member.Mail,
			Phone:       member.Phone,
			PhoneParent: member.PhoneParent,
			Birthday:    member.Birthday,
			Returning:   member.Returning,
		}
		msg := c.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:member.updated"))
		msg.SetBody(body)
		meta := messages.Metadata{Producer: c.clientID}
		meta.RequestHeaders.Set(r.Header)
		msg.SetMeta(&meta)
		c.publisher.Publish(msg)
	}
	for memberID, _ := range memberIDs {
		msg := c.publisher.MessageFunc()(streaminterface.SubjectFromStr("nathejk:member.deleted"))
		msg.SetBody(messages.NathejkMemberDeleted{
			MemberID: memberID,
			TeamID:   req.TeamID,
		})
		meta := messages.Metadata{Producer: c.clientID}
		meta.RequestHeaders.Set(r.Header)
		msg.SetMeta(&meta)
		c.publisher.Publish(msg)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(TeamUpdateResponse{
		OK: err == nil,
	})
}

type TeamReadResponse struct {
	TeamID            types.TeamID      `json:"teamId"`
	Type              types.Enum        `json:"type"`
	Name              string            `json:"name"`
	GroupName         string            `json:"groupName"`
	Korps             string            `json:"korps"`
	AdvspejdNumber    string            `json:"advspejdNumber"`
	ContactName       string            `json:"contactName"`
	ContactAddress    string            `json:"contactAddress"`
	ContactPostalCode string            `json:"contactPostalCode"`
	ContactEmail      types.Email       `json:"contactMail"`
	ContactPhone      types.PhoneNumber `json:"contactPhone"`
	ContactRole       string            `json:"contactRole"`
	Members           []struct {
		MemberID    types.MemberID    `json:"memberId"`
		Name        string            `json:"name"`
		Address     string            `json:"address"`
		PostalCode  string            `json:"postalCode"`
		Mail        types.Email       `json:"mail"`
		Phone       types.PhoneNumber `json:"phone"`
		PhoneParent types.PhoneNumber `json:"phoneParent"`
		Birthday    string            `json:"birthday"`
		Returning   bool              `json:"returning"`
	} `json:"members"`
}

func (c *apiController) ApiTeamReadHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("ApiTeamReadHandler %q", r.URL.String())
	path := strings.Split(r.URL.Path[1:], "/")
	if len(path) != 3 {
		log.Printf("invalid request %q", path)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	teamID := path[2]
	team, err := c.redisclient.HGet("teams", string(teamID)).Result()
	if err != nil {
		log.Printf("NOT FOUND")
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	var response TeamReadResponse
	json.Unmarshal([]byte(team), &response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
*/
