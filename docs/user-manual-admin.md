# Administrator manual

## Signing in
Open the site and press **Sign in**. The first administrator is created from `ADMIN_EMAIL` / `ADMIN_PASSWORD` when the server starts; change the password under **Account** right away. After 10 wrong attempts in a minute the sign-in is blocked for a short time.

## Daily tasks
| Task | Where | Steps |
|---|---|---|
| Add a subject | Subjects | Enter name and unique code, **Add**. |
| Add a teacher | Teachers | Fill in e-mail, temporary password, name, department; then tick the subjects the teacher may teach and **Save subjects**. |
| Add a class | Classes | Name, level, section, homeroom teacher, capacity. |
| Add a student | Students | Choose the class, enter the student number (unique) and account data. |
| Add a parent and link a child | Parents | Create the parent, then use **Link** to attach one or more students. |
| Build the timetable | Timetable | Choose class, subject, teacher, day, start/end, room. The system refuses a lesson when the class, teacher or room is busy at that time, or when the teacher does not teach the subject. |
| Disable or re-enable an account | Users | **Toggle** next to the user. Disabled users cannot sign in; their history is kept. |
| Reset a password | Users | Enter a new temporary password for the user. |
| Publish news | News | Title, text, optionally **Pin to the top**. News appear on the public home page. Text is shown as plain text (no HTML). |
| Send an announcement | Announce | Choose the audience (everyone or one role), title and text. Recipients see it in their notifications. |
| Download a report | Gradebook, Attendance statistics, a student's marks or attendance | Press **Export CSV**. Open it in Excel or LibreOffice. |
| Run the attendance reminder now | Dashboard, reminders button | Creates reminders for teachers who have not marked today's attendance (the system does this automatically at 15:00). |

## Good practice
* Give new users a temporary password and ask them to change it.
* Deactivate instead of deleting when a person leaves: marks and attendance stay available for reports.
* Back up the database daily (see `operations.md`).
