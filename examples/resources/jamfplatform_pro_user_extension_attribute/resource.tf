# Text field user extension attribute. User EAs support Text Field and Pop-up
# Menu input types only.
resource "jamfplatform_pro_user_extension_attribute" "employee_id" {
  name        = "Employee ID"
  description = "HR-issued employee identifier."
  data_type   = "String"
  input_type  = "Text Field"
}

# Pop-up menu user extension attribute with an ordered list of choices.
resource "jamfplatform_pro_user_extension_attribute" "cost_center" {
  name               = "Cost Center"
  data_type          = "String"
  input_type         = "Pop-up Menu"
  popup_menu_choices = ["Engineering", "Sales", "Operations"]
}
