package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/mzviagintsev/school-management/internal/models"
)

// ---------- users ----------

func (h *Handler) adminUsersPage(w http.ResponseWriter, r *http.Request, p *Page) {
	q := r.URL.Query()
	users, err := h.svc.Users.List(r.Context(), models.UserRole(q.Get("role")), q.Get("q"), 100, 0)
	if err != nil && p.Error == "" {
		p.Error = errMessage(err)
	}
	p.Data = map[string]any{"Users": users, "Role": q.Get("role"), "Q": q.Get("q"), "Me": currentUser(r).ID}
	// htmx live search only needs the table
	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Target") == "user-table" {
		h.render(w, "admin_users", "user_table", p, http.StatusOK)
		return
	}
	h.page(w, "admin_users", p)
}

func (h *Handler) adminUsers(w http.ResponseWriter, r *http.Request) {
	h.adminUsersPage(w, r, h.newPage(r, "Users", "users"))
}

// adminUserCreate creates administrator accounts; teachers, students and parents
// have their own pages because they also need a profile.
func (h *Handler) adminUserCreate(w http.ResponseWriter, r *http.Request) {
	_, err := h.svc.Auth.Register(r.Context(), &models.RegisterRequest{
		Email: strings.TrimSpace(r.FormValue("email")), Password: r.FormValue("password"),
		FirstName: strings.TrimSpace(r.FormValue("first_name")), LastName: strings.TrimSpace(r.FormValue("last_name")),
		Role: models.RoleAdmin,
	})
	if err != nil {
		p := h.newPage(r, "Users", "users")
		p.Error = errMessage(err)
		h.adminUsersPage(w, r, p)
		return
	}
	redirect(w, r, "/admin/users", "Administrator created")
}

func (h *Handler) adminUserToggle(w http.ResponseWriter, r *http.Request) {
	id := pathInt(r, "id")
	u, err := h.svc.Users.GetByID(r.Context(), id)
	if err == nil {
		active := !u.IsActive
		_, err = h.svc.Users.Update(r.Context(), actorOf(r), id, &models.UpdateUserRequest{IsActive: &active})
	}
	if err != nil {
		p := h.newPage(r, "Users", "users")
		p.Error = errMessage(err)
		h.adminUsersPage(w, r, p)
		return
	}
	redirect(w, r, "/admin/users?q="+r.FormValue("q")+"&role="+r.FormValue("role"), "Account updated")
}

func (h *Handler) adminUserPassword(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Users.ResetPassword(r.Context(), pathInt(r, "id"), r.FormValue("new_password")); err != nil {
		p := h.newPage(r, "Users", "users")
		p.Error = errMessage(err)
		h.adminUsersPage(w, r, p)
		return
	}
	redirect(w, r, "/admin/users", "Password reset")
}

// ---------- classes ----------

type classRow struct {
	Class    *models.Class
	Teacher  string
	Students int
}

func (h *Handler) adminClassesPage(w http.ResponseWriter, r *http.Request, p *Page) {
	ctx := r.Context()
	classes := h.sortedClasses(ctx)
	teachers, _ := h.svc.Teachers.List(ctx, 100, 0)
	names := map[int64]string{}
	for _, t := range teachers {
		names[t.UserID] = strings.TrimSpace(t.FirstName + " " + t.LastName)
	}
	rows := make([]classRow, 0, len(classes))
	for _, c := range classes {
		students, _ := h.svc.Students.ListByClass(ctx, c.ID, 100, 0)
		rows = append(rows, classRow{Class: c, Teacher: names[c.TeacherID], Students: len(students)})
	}
	p.Data = map[string]any{"Rows": rows, "Teachers": teachers}
	h.page(w, "admin_classes", p)
}

func (h *Handler) adminClasses(w http.ResponseWriter, r *http.Request) {
	h.adminClassesPage(w, r, h.newPage(r, "Classes", "classes"))
}

func (h *Handler) adminClassCreate(w http.ResponseWriter, r *http.Request) {
	maxStudents, _ := strconv.Atoi(r.FormValue("max_students"))
	_, err := h.svc.Classes.Create(r.Context(), &models.CreateClassRequest{
		Name: strings.TrimSpace(r.FormValue("name")), Level: strings.TrimSpace(r.FormValue("level")),
		Section: strings.TrimSpace(r.FormValue("section")), TeacherID: formInt(r, "teacher_id"), MaxStudents: maxStudents,
	})
	if err != nil {
		p := h.newPage(r, "Classes", "classes")
		p.Error = errMessage(err)
		h.adminClassesPage(w, r, p)
		return
	}
	redirect(w, r, "/admin/classes", "Class created")
}

func (h *Handler) adminClassDelete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Classes.Delete(r.Context(), pathInt(r, "id")); err != nil {
		p := h.newPage(r, "Classes", "classes")
		p.Error = "This class still has students or records, so it can't be deleted. Move its students first."
		h.adminClassesPage(w, r, p)
		return
	}
	redirect(w, r, "/admin/classes", "Class deleted")
}

// ---------- subjects ----------

func (h *Handler) adminSubjectsPage(w http.ResponseWriter, r *http.Request, p *Page) {
	subjects, _ := h.svc.Subjects.List(r.Context(), 100, 0)
	p.Data = subjects
	h.page(w, "admin_subjects", p)
}

func (h *Handler) adminSubjects(w http.ResponseWriter, r *http.Request) {
	h.adminSubjectsPage(w, r, h.newPage(r, "Subjects", "subjects"))
}

func (h *Handler) adminSubjectCreate(w http.ResponseWriter, r *http.Request) {
	credits, _ := strconv.Atoi(r.FormValue("credits"))
	_, err := h.svc.Subjects.Create(r.Context(), &models.CreateSubjectRequest{
		Name: r.FormValue("name"), Code: r.FormValue("code"), Description: r.FormValue("description"), Credits: credits,
	})
	if err != nil {
		p := h.newPage(r, "Subjects", "subjects")
		p.Error = errMessage(err)
		h.adminSubjectsPage(w, r, p)
		return
	}
	redirect(w, r, "/admin/subjects", "Subject added")
}

func (h *Handler) adminSubjectDelete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Subjects.Delete(r.Context(), pathInt(r, "id")); err != nil {
		p := h.newPage(r, "Subjects", "subjects")
		p.Error = "This subject is used by grades or the timetable, so it can't be deleted."
		h.adminSubjectsPage(w, r, p)
		return
	}
	redirect(w, r, "/admin/subjects", "Subject deleted")
}

// ---------- teachers ----------

type teacherRow struct {
	Teacher  *models.TeacherWithUser
	Subjects map[int64]bool
	Names    []string
}

func (h *Handler) adminTeachersPage(w http.ResponseWriter, r *http.Request, p *Page) {
	ctx := r.Context()
	teachers, _ := h.svc.Teachers.List(ctx, 100, 0)
	subjects, _ := h.svc.Subjects.List(ctx, 100, 0)
	rows := make([]teacherRow, 0, len(teachers))
	for _, t := range teachers {
		row := teacherRow{Teacher: t, Subjects: map[int64]bool{}}
		assigned, _ := h.svc.Subjects.TeacherSubjects(ctx, t.ID)
		for _, s := range assigned {
			row.Subjects[s.ID] = true
			row.Names = append(row.Names, s.Name)
		}
		rows = append(rows, row)
	}
	p.Data = map[string]any{"Rows": rows, "Subjects": subjects}
	h.page(w, "admin_teachers", p)
}

func (h *Handler) adminTeachers(w http.ResponseWriter, r *http.Request) {
	h.adminTeachersPage(w, r, h.newPage(r, "Teachers", "teachers"))
}

func (h *Handler) adminTeacherCreate(w http.ResponseWriter, r *http.Request) {
	_, err := h.svc.Teachers.Create(r.Context(), &models.CreateTeacherRequest{
		Email: strings.TrimSpace(r.FormValue("email")), Password: r.FormValue("password"),
		FirstName: strings.TrimSpace(r.FormValue("first_name")), LastName: strings.TrimSpace(r.FormValue("last_name")),
		Department: r.FormValue("department"), Specialization: r.FormValue("specialization"), HireDate: r.FormValue("hire_date"),
	})
	if err != nil {
		p := h.newPage(r, "Teachers", "teachers")
		p.Error = errMessage(err)
		h.adminTeachersPage(w, r, p)
		return
	}
	redirect(w, r, "/admin/teachers", "Teacher added")
}

func (h *Handler) adminTeacherSubjects(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	var ids []int64
	for _, v := range r.Form["subject"] {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	if _, err := h.svc.Subjects.AssignToTeacher(r.Context(), pathInt(r, "id"), ids); err != nil {
		p := h.newPage(r, "Teachers", "teachers")
		p.Error = errMessage(err)
		h.adminTeachersPage(w, r, p)
		return
	}
	redirect(w, r, "/admin/teachers", "Subjects updated")
}

// ---------- students ----------

func (h *Handler) adminStudentsPage(w http.ResponseWriter, r *http.Request, p *Page) {
	ctx := r.Context()
	classes := h.sortedClasses(ctx)
	classID := queryInt(r, "class")
	var students []*models.StudentWithUser
	if classID > 0 {
		students, _ = h.svc.Students.ListByClass(ctx, classID, 100, 0)
	} else {
		students, _ = h.svc.Students.List(ctx, 100, 0)
	}
	p.Data = map[string]any{"Students": students, "Classes": classes, "ClassID": classID, "ClassNames": h.classNames(ctx)}
	h.page(w, "admin_students", p)
}

func (h *Handler) adminStudents(w http.ResponseWriter, r *http.Request) {
	h.adminStudentsPage(w, r, h.newPage(r, "Students", "students"))
}

func (h *Handler) adminStudentCreate(w http.ResponseWriter, r *http.Request) {
	_, err := h.svc.Students.Create(r.Context(), &models.CreateStudentRequest{
		Email: strings.TrimSpace(r.FormValue("email")), Password: r.FormValue("password"),
		FirstName: strings.TrimSpace(r.FormValue("first_name")), LastName: strings.TrimSpace(r.FormValue("last_name")),
		ClassID: formInt(r, "class_id"), StudentNumber: strings.TrimSpace(r.FormValue("student_number")),
		EnrollmentDate: r.FormValue("enrollment_date"),
	})
	if err != nil {
		p := h.newPage(r, "Students", "students")
		p.Error = errMessage(err)
		h.adminStudentsPage(w, r, p)
		return
	}
	redirect(w, r, "/admin/students", "Student enrolled")
}

// ---------- parents ----------

func (h *Handler) adminParentsPage(w http.ResponseWriter, r *http.Request, p *Page) {
	ctx := r.Context()
	parents, _ := h.svc.Parents.List(ctx, 200, 0)
	students, _ := h.svc.Students.List(ctx, 100, 0)
	p.Data = map[string]any{"Parents": parents, "Students": students}
	h.page(w, "admin_parents", p)
}

func (h *Handler) adminParents(w http.ResponseWriter, r *http.Request) {
	h.adminParentsPage(w, r, h.newPage(r, "Parents", "parents"))
}

func (h *Handler) adminParentCreate(w http.ResponseWriter, r *http.Request) {
	parent, err := h.svc.Parents.Create(r.Context(), &models.CreateParentRequest{
		Email: strings.TrimSpace(r.FormValue("email")), Password: r.FormValue("password"),
		FirstName: strings.TrimSpace(r.FormValue("first_name")), LastName: strings.TrimSpace(r.FormValue("last_name")),
		Relationship: r.FormValue("relationship"), PhoneNumber: r.FormValue("phone_number"),
	})
	if err == nil {
		if sid := formInt(r, "student_id"); sid > 0 {
			err = h.svc.Parents.LinkStudent(r.Context(), parent.ID, sid)
		}
	}
	if err != nil {
		p := h.newPage(r, "Parents", "parents")
		p.Error = errMessage(err)
		h.adminParentsPage(w, r, p)
		return
	}
	redirect(w, r, "/admin/parents", "Parent added")
}

func (h *Handler) adminParentLink(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Parents.LinkStudent(r.Context(), formInt(r, "parent_id"), formInt(r, "student_id")); err != nil {
		p := h.newPage(r, "Parents", "parents")
		p.Error = errMessage(err)
		h.adminParentsPage(w, r, p)
		return
	}
	redirect(w, r, "/admin/parents", "Parent linked to student")
}

// ---------- timetable ----------

func (h *Handler) adminTimetablePage(w http.ResponseWriter, r *http.Request, p *Page, classID int64) {
	ctx := r.Context()
	classes := h.sortedClasses(ctx)
	class := pickClass(classes, classID)
	data := map[string]any{"Classes": classes, "Class": class}
	data["Subjects"], _ = h.svc.Subjects.List(ctx, 100, 0)
	data["Teachers"], _ = h.svc.Teachers.List(ctx, 100, 0)
	data["Form"] = r.Form
	if class != nil {
		items, _ := h.svc.Timetable.ListByClass(ctx, class.ID)
		data["Week"] = buildWeek(items, p.Today)
	}
	p.Data = data
	h.page(w, "admin_timetable", p)
}

func (h *Handler) adminTimetable(w http.ResponseWriter, r *http.Request) {
	h.adminTimetablePage(w, r, h.newPage(r, "Timetable", "timetable"), queryInt(r, "class"))
}

func (h *Handler) adminTimetableCreate(w http.ResponseWriter, r *http.Request) {
	day, _ := strconv.Atoi(r.FormValue("day_of_week"))
	classID := formInt(r, "class_id")
	_, err := h.svc.Timetable.Create(r.Context(), &models.TimetableRequest{
		ClassID: classID, SubjectID: formInt(r, "subject_id"), TeacherID: formInt(r, "teacher_id"),
		DayOfWeek: day, StartTime: r.FormValue("start_time"), EndTime: r.FormValue("end_time"),
		RoomNumber: r.FormValue("room_number"),
	})
	if err != nil {
		p := h.newPage(r, "Timetable", "timetable")
		p.Error = errMessage(err)
		h.adminTimetablePage(w, r, p, classID)
		return
	}
	redirect(w, r, fmt.Sprintf("/admin/timetable?class=%d", classID), "Lesson added")
}

func (h *Handler) adminTimetableDelete(w http.ResponseWriter, r *http.Request) {
	_ = h.svc.Timetable.Delete(r.Context(), pathInt(r, "id"))
	redirect(w, r, fmt.Sprintf("/admin/timetable?class=%d", formInt(r, "class_id")), "Lesson removed")
}

// ---------- announcements & reminders ----------

func (h *Handler) adminAnnounce(w http.ResponseWriter, r *http.Request) {
	h.page(w, "admin_announce", h.newPage(r, "Announcements", "announce"))
}

func (h *Handler) adminAnnounceSend(w http.ResponseWriter, r *http.Request) {
	n, err := h.svc.Notifications.Announce(r.Context(), &models.AnnouncementRequest{
		Role: models.UserRole(r.FormValue("role")), Title: r.FormValue("title"), Message: r.FormValue("message"),
	})
	if err != nil {
		p := h.newPage(r, "Announcements", "announce")
		p.Error = errMessage(err)
		h.page(w, "admin_announce", p)
		return
	}
	redirect(w, r, "/admin/announce", fmt.Sprintf("Announcement sent to %d people", n))
}

func (h *Handler) adminRunReminders(w http.ResponseWriter, r *http.Request) {
	n, err := h.svc.Reminders.RunOnce(r.Context())
	msg := fmt.Sprintf("Sent %d attendance reminders", n)
	if err != nil {
		msg = errMessage(err)
	}
	redirect(w, r, "/", msg)
}
