require 'xcodeproj'
scheme = ENV.fetch('APPLE_APP_SCHEME')
project_path = Dir.glob('apps/mobile/ios/*.xcodeproj').find { |p| File.basename(p, '.xcodeproj') == scheme }
raise 'Application project missing' unless project_path
project = Xcodeproj::Project.open(project_path)
target = project.targets.find { |candidate| candidate.name == scheme }
raise 'Application target missing' unless target
configuration = target.build_configurations.find { |candidate| candidate.name == 'Release' }
raise 'Release configuration missing' unless configuration
configuration.build_settings['CODE_SIGN_STYLE'] = 'Manual'
configuration.build_settings['CODE_SIGN_IDENTITY'] = 'Apple Distribution'
configuration.build_settings['DEVELOPMENT_TEAM'] = ENV.fetch('APPLE_TEAM_ID')
configuration.build_settings['PROVISIONING_PROFILE_SPECIFIER'] = ENV.fetch('APPLE_PROVISIONING_PROFILE_NAME')
project.save
