Pod::Spec.new do |spec|
  spec.name         = 'Gprobe'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/probechain/go-probe'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'iOS Probeum Client'
  spec.source       = { :git => 'https://github.com/probechain/go-probe.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Gprobe.framework'

	spec.prepare_command = <<-CMD
    curl https://gprobestore.blob.core.windows.net/builds/{{.Archive}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv {{.Archive}}/Gprobe.framework Frameworks
    rm -rf {{.Archive}}
  CMD
end
