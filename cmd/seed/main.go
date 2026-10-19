// Command seed fills an empty database with a small demo school: subjects, teachers,
// two classes with students, parents, a weekly timetable, three weeks of attendance
// and first-term grades. Everything goes through the service layer, so all business
// rules (validation, conflict detection, notifications) apply exactly as in the app.
//
//	DATABASE_URL=... go run ./cmd/seed
//
// The bootstrap admin (ADMIN_EMAIL / ADMIN_PASSWORD) is created if missing.
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/mzviagintsev/school-management/internal/db"
	"github.com/mzviagintsev/school-management/internal/models"
	"github.com/mzviagintsev/school-management/internal/repository"
	"github.com/mzviagintsev/school-management/internal/services"
)

const (
	teacherPassword = "teacher123"
	studentPassword = "student123"
	parentPassword  = "parent123"
)

func must[T any](v T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return v
}

func email(first, last string) string {
	return strings.ToLower(first+"."+last) + "@school.test"
}

func main() {
	ctx := context.Background()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/school_management?sslmode=disable"
	}
	database := must(db.New(dbURL))
	defer database.Close()
	conn := database.Conn()

	userRepo := repository.NewUserRepository(conn)
	classRepo := repository.NewClassRepository(conn)
	teacherRepo := repository.NewTeacherRepository(conn)
	studentRepo := repository.NewStudentRepository(conn)
	subjectRepo := repository.NewSubjectRepository(conn)
	parentRepo := repository.NewParentRepository(conn)
	attendanceRepo := repository.NewAttendanceRepository(conn)
	gradeRepo := repository.NewGradeRepository(conn)
	timetableRepo := repository.NewTimetableRepository(conn)
	notificationRepo := repository.NewNotificationRepository(conn)

	users := services.NewUserService(userRepo)
	subjects := services.NewSubjectService(subjectRepo, teacherRepo)
	teachers := services.NewTeacherService(teacherRepo, userRepo)
	classes := services.NewClassService(classRepo, userRepo)
	students := services.NewStudentService(studentRepo, classRepo, userRepo)
	parents := services.NewParentService(parentRepo, studentRepo, userRepo)
	access := services.NewAccessControl(classRepo, teacherRepo, parentRepo, timetableRepo)
	notifier := services.NewNotificationService(notificationRepo, parentRepo)
	attendance := services.NewAttendanceService(attendanceRepo, studentRepo, classRepo, access, notifier)
	grades := services.NewGradeService(gradeRepo, studentRepo, subjectRepo, access, notifier)
	newsRepo := repository.NewNewsRepository(conn)
	news := services.NewNewsService(newsRepo)
	timetable := services.NewTimetableService(timetableRepo, classRepo, subjectRepo, teacherRepo, studentRepo)

	// Make sure the demo admin login exists. EnsureAdmin alone is not enough: it only
	// creates an admin when there is none, so a database that already had another admin
	// (e.g. from earlier testing) would never get this account.
	adminEmail := os.Getenv("ADMIN_EMAIL")
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminEmail == "" {
		adminEmail, adminPassword = "admin@school.test", "admin12345"
	}
	adminUser := must(userRepo.GetByEmail(ctx, adminEmail))
	switch {
	case adminUser == nil:
		auth := services.NewAuthService(userRepo, "")
		adminUser = must(auth.Register(ctx, &models.RegisterRequest{
			Email: adminEmail, Password: adminPassword, FirstName: "Administrator", Role: models.RoleAdmin,
		}))
		fmt.Printf("Created administrator %s\n", adminEmail)
	case adminUser.Role != models.RoleAdmin:
		log.Fatalf("%s already exists but is a %s, not an admin; set ADMIN_EMAIL to another address", adminEmail, adminUser.Role)
	default:
		// Existing admin account: reset its password so the printed credentials are guaranteed to work
		if err := users.ResetPassword(ctx, adminUser.ID, adminPassword); err != nil {
			log.Fatal(err)
		}
		if !adminUser.IsActive {
			active := true
			must(users.Update(ctx, services.Actor{}, adminUser.ID, &models.UpdateUserRequest{IsActive: &active}))
		}
	}
	admin := services.Actor{UserID: adminUser.ID, Role: models.RoleAdmin}

	if existing := must(subjects.List(ctx, 1, 0)); len(existing) > 0 {
		fmt.Println("The database already contains school data, so no demo data was added.")
		fmt.Printf("Administrator login is ready: %s / %s\n", adminEmail, adminPassword)
		return
	}

	// ---- subjects: index i and i+3 belong to different teachers (see timetable below)
	subjectDefs := []struct {
		name, code string
		credits    int
	}{
		{"Mathematics", "MATH", 5}, {"Physics", "PHYS", 3}, {"Literature", "LIT", 4},
		{"History", "HIST", 3}, {"Biology", "BIO", 3}, {"English", "ENG", 4},
	}
	var subs []*models.Subject
	for _, d := range subjectDefs {
		subs = append(subs, must(subjects.Create(ctx, &models.CreateSubjectRequest{Name: d.name, Code: d.code, Credits: d.credits})))
	}

	// ---- teachers, each qualified for two subjects
	teacherDefs := []struct {
		first, last, dept string
		subjects          []int
	}{
		{"Anna", "Kovacs", "Sciences", []int{0, 1}},
		{"Peter", "Novak", "Humanities", []int{2, 3}},
		{"Maria", "Szabo", "Languages", []int{4, 5}},
	}
	var tch []*models.TeacherWithUser
	owner := map[int]int{} // subject index -> teacher index
	for i, d := range teacherDefs {
		t := must(teachers.Create(ctx, &models.CreateTeacherRequest{
			Email: email(d.first, d.last), Password: teacherPassword, FirstName: d.first, LastName: d.last,
			Department: d.dept, HireDate: "2019-09-01",
		}))
		var ids []int64
		for _, si := range d.subjects {
			ids = append(ids, subs[si].ID)
			owner[si] = i
		}
		must(subjects.AssignToTeacher(ctx, t.ID, ids))
		tch = append(tch, t)
	}

	// ---- classes (homeroom: Kovacs for 9A, Novak for 9B)
	classA := must(classes.Create(ctx, &models.CreateClassRequest{Name: "9A", Level: "9", Section: "A", TeacherID: tch[0].UserID, MaxStudents: 28}))
	classB := must(classes.Create(ctx, &models.CreateClassRequest{Name: "9B", Level: "9", Section: "B", TeacherID: tch[1].UserID, MaxStudents: 28}))

	// ---- students
	names := [][2]string{
		{"Bence", "Toth"}, {"Lili", "Nagy"}, {"David", "Horvath"}, {"Emma", "Varga"}, {"Mark", "Kiss"},
		{"Sofia", "Molnar"}, {"Adam", "Farkas"}, {"Hanna", "Balogh"},
		{"Mate", "Papp"}, {"Zoe", "Takacs"}, {"Levente", "Juhasz"}, {"Anna", "Meszaros"}, {"Daniel", "Olah"},
		{"Nora", "Simon"}, {"Oliver", "Racz"}, {"Lara", "Fekete"},
	}
	var all []*models.StudentWithUser
	for i, n := range names {
		class := classA
		if i >= 8 {
			class = classB
		}
		all = append(all, must(students.Create(ctx, &models.CreateStudentRequest{
			Email: email(n[0], n[1]), Password: studentPassword, FirstName: n[0], LastName: n[1],
			ClassID: class.ID, StudentNumber: fmt.Sprintf("2026-%03d", i+1), EnrollmentDate: "2026-09-01",
		})))
	}

	// ---- parents
	parentDefs := []struct {
		first, last, rel string
		children         []int
	}{
		{"Eva", "Toth", "mother", []int{0}},
		{"Gabor", "Papp", "father", []int{8}},
		{"Judit", "Nagy", "mother", []int{1}},
	}
	for _, d := range parentDefs {
		p := must(parents.Create(ctx, &models.CreateParentRequest{
			Email: email(d.first, d.last), Password: parentPassword, FirstName: d.first, LastName: d.last,
			Relationship: d.rel, PhoneNumber: "+36 30 123 4567",
		}))
		for _, c := range d.children {
			if err := parents.LinkStudent(ctx, p.ID, all[c].ID); err != nil {
				log.Fatal(err)
			}
		}
	}

	// ---- timetable: Mon-Fri, four periods. 9A takes subject i, 9B takes subject i+3,
	// so the two classes never need the same teacher at the same time.
	periods := [][2]string{{"08:00", "08:45"}, {"08:55", "09:40"}, {"09:50", "10:35"}, {"10:45", "11:30"}}
	for day := 1; day <= 5; day++ {
		for p, slot := range periods {
			ia := (day*4 + p) % 6
			ib := (ia + 3) % 6
			for _, x := range []struct {
				class *models.Class
				si    int
				room  string
			}{{classA, ia, "101"}, {classB, ib, "102"}} {
				must(timetable.Create(ctx, &models.TimetableRequest{
					ClassID: x.class.ID, SubjectID: subs[x.si].ID, TeacherID: tch[owner[x.si]].ID,
					DayOfWeek: day, StartTime: slot[0], EndTime: slot[1], RoomNumber: x.room,
				}))
			}
		}
	}

	// ---- attendance for the last 15 school days (today is left open to show reminders)
	rng := rand.New(rand.NewSource(42))
	today := time.Now()
	var days []time.Time
	for d := today.AddDate(0, 0, -1); len(days) < 15; d = d.AddDate(0, 0, -1) {
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			days = append([]time.Time{d}, days...)
		}
	}
	for _, day := range days {
		for _, class := range []*models.Class{classA, classB} {
			req := &models.MarkAttendanceRequest{ClassID: class.ID, Date: day.Format("2006-01-02")}
			for i, s := range all {
				if s.ClassID != class.ID {
					continue
				}
				status := models.AttendancePresent
				r := rng.Float64()
				switch {
				case i == 10 && r < 0.35: // one pupil with poor attendance -> low-attendance warning
					status = models.AttendanceAbsent
				case r < 0.04:
					status = models.AttendanceAbsent
				case r < 0.09:
					status = models.AttendanceLate
				case r < 0.11:
					status = models.AttendanceExcused
				}
				req.Records = append(req.Records, models.AttendanceEntry{StudentID: s.ID, Status: status})
			}
			must(attendance.MarkAttendance(ctx, admin, req))
		}
	}

	// ---- first-term grades (five-point scale): every pupil has an "ability" level, marks vary around it
	for _, s := range all {
		ability := 3.0 + rng.Float64()*1.8 // 3.0 .. 4.8
		for _, sub := range subs {
			for k := 0; k < 2+rng.Intn(3); k++ {
				m := math.Round(ability + rng.NormFloat64()*0.7)
				m = max(2, min(5, m))
				g := must(grades.Record(ctx, admin, &models.RecordGradeRequest{StudentID: s.ID, SubjectID: sub.ID, Marks: m, Term: "Q1"}))
				// spread the marks over the last school days so the weekly diary has something to show
				when := days[rng.Intn(len(days))]
				when = time.Date(when.Year(), when.Month(), when.Day(), 9+rng.Intn(6), rng.Intn(60), 0, 0, time.Local)
				must(conn.ExecContext(ctx, `UPDATE grades SET created_at = $1, updated_at = $1 WHERE id = $2`, when, g.ID))
			}
		}
	}

	// ---- school news for the home page
	for i, n := range []models.NewsRequest{
		{Title: "Welcome to the new school year", Pinned: true, Body: "Dear students and parents,\n\nThe school year starts with a new e-diary. Marks now use the five-point scale, and the weekly diary shows lessons together with the marks received each day.\n\nPlease check your account and report any problems to the school office."},
		{Title: "Parent-teacher meeting on 22 October", Body: "Teachers will be available from 16:00 to 19:00 in their classrooms. Please bring the diary with you."},
		{Title: "Autumn break", Body: "Lessons pause from 2 to 6 November. The first day back is Monday, 9 November."},
		{Title: "School library: new books", Body: "Fifty new titles for grades 7-9 have arrived, including literature from the reading list for this term."},
	} {
		n := n
		item := must(news.Create(ctx, admin, &n))
		when := time.Now().AddDate(0, 0, -i*3)
		must(conn.ExecContext(ctx, `UPDATE news SET created_at = $1, updated_at = $1 WHERE id = $2`, when, item.ID))
	}

	fmt.Println("Demo school created.")
	fmt.Printf("  admin    %s / %s\n", adminEmail, adminPassword)
	fmt.Printf("  teacher  %s / %s  (homeroom 9A)\n", email("Anna", "Kovacs"), teacherPassword)
	fmt.Printf("  student  %s / %s  (9A)\n", email("Bence", "Toth"), studentPassword)
	fmt.Printf("  parent   %s / %s  (mother of Bence Toth)\n", email("Eva", "Toth"), parentPassword)
}
