resource "jamfplatform_pro_login_page_settings" "this" {
  # Show the custom disclaimer on the Jamf Pro login page.
  include_custom_disclaimer = true

  # The three text fields are required on every write, even when
  # include_custom_disclaimer is false. Heading and action are capped at 20
  # characters; the main body is capped at 2,500.
  disclaimer_heading   = "Authorized Use Only"
  disclaimer_main_text = "By signing in you agree to the organization's acceptable use policy."
  action_text          = "I Agree"
}
