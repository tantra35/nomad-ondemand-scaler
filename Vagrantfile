Vagrant.configure("2") do |config|
	config.vm.box = "ubuntu/jammy64"

	config.vm.provider "virtualbox" do |vb|
		vb.memory = "8192"
		vb.cpus = 2
		vb.customize ["modifyvm", :id, "--natdnshostresolver1", "on"]
	end

	config.vm.define "nomad-ondemand-scaler" do |node|
		node.vm.hostname = "nomad-ondemand-scaler"
		node.vm.synced_folder ".builddocker/", "/tmp/.builddocker"
		node.vm.synced_folder "./", "/home/vagrant/builddocker"

		node.vm.provision :salt do |salt|
			salt.masterless = true
			salt.colorize = true
			salt.verbose = true
			salt.log_level = "info"
			salt.install_type = "stable"
			salt.version = "3005"
			salt.run_highstate = true
			salt.salt_call_args = ["--file-root", "/tmp/.builddocker"]

			salt.pillar({})
		end

		node.vm.provision "shell", inline: <<-SCRIPT
			if ! grep "cd /home/vagrant/builddocker" /home/vagrant/.profile ; then
				echo 'cd /home/vagrant/builddocker' >> /home/vagrant/.profile
				echo 'export MYLIBSPATH=$HOME/lib' >> /home/vagrant/.profile
				echo 'export MYTOOLSPATH=/opt/' >> /home/vagrant/.profile
			fi
		SCRIPT
	end
end
