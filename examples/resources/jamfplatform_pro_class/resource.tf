# A Jamf Pro class for Apple Classroom — groups students and teachers (by
# username) and student/teacher/mobile-device groups (by ID). Membership is
# authoritative: each set is applied in full on every change, so removing an
# entry removes the member and omitting a set leaves it empty.
resource "jamfplatform_pro_class" "biology_101" {
  name        = "Biology 101"
  description = "Year 9 biology, room 204"

  # Students and teachers are referenced by username. Jamf Pro resolves each to
  # a user record (creating one if the username is not yet known) and echoes the
  # resolved IDs in the read-only student_ids / teacher_ids attributes.
  students = [
    "ada.lovelace@school.example",
    "alan.turing@school.example",
  ]
  teachers = [
    "grace.hopper@school.example",
  ]
}

# A class whose membership is driven entirely by groups, scoped to a site.
resource "jamfplatform_pro_class" "all_year_9" {
  name    = "All Year 9"
  site_id = "1"

  # Reference existing Jamf Pro user groups and a mobile device group by ID.
  # The class's device list is derived by Jamf Pro from these groups' members.
  student_group_ids       = ["3"]
  teacher_group_ids       = ["1"]
  mobile_device_group_ids = ["66"]
}
