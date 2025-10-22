run "test_data_sources" {
  command = plan
  module {
      source = "./data_sources"
  }
}

run "test_resources" {
  command = apply
  module {
      source = "./resources"
  }
}
