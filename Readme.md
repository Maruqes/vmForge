# vmForge
Turn your home server on a virtual machine launcher


Manage multiple virtual machines in the same place

This project's idea is to simplify the creation of docker containers and access them with ssh/sshfs. With this project there will be no "test" folders on your computer, infinite unorganized systems running or even the need to create a "base project" as you can create images for them and access everything anywhere.

Basically a virtual machine creator but ULTRA simplistic


## Deployment

### Linux

```
sudo apt install haproxy
apt install sqlite3
apt install sqlite-utils
```

DO NOT FORGET TO INSTALL DOCKER


## Features

- Creation of docker images Ubuntu images
- Create custom commands for those images
- Creation of containers based on the images
- Auth system with admin accounts

## Login screen / Auth system
In this screens you can login / create / delete admins
![Screenshot from 2024-09-23 19-29-52](https://github.com/user-attachments/assets/86c038f5-9ecd-4698-bc80-c1a378ea9df8)
![Screenshot from 2024-09-23 19-29-42](https://github.com/user-attachments/assets/dcd218af-18c5-43b9-abd2-f1f9609d60e9)


## Creating an image that will come with Nano installed
![Screenshot from 2024-09-23 19-29-03](https://github.com/user-attachments/assets/02be1f97-a5a2-42cc-937a-7ea7baed44fc)

## Creating a container based on that image
![Screenshot from 2024-09-23 19-29-19](https://github.com/user-attachments/assets/cde1a1e1-5e3e-4c5b-a1f5-3465dcef4f23)

## What you will see
You have 2 buttons for ssh/sshfs to enter and mount the VM on your machine

Start/stop/restart/delete
![Screenshot from 2024-09-23 19-30-12](https://github.com/user-attachments/assets/f0b7698f-47b3-4825-8674-7b5d622dff3c)


### READ THIS
DO NOT USE THIS PROJECT FOR OTHER THAN PERSONAL USE, IT IS NOT PROGRAMMED TO ONLY CONTROL VM_FORGE CONTAINERS, IF YOUR ACCOUNT IS IN ANY WAY ACCESSED BY UNAUTHORIZED PERSONA THEY CAN START/STOP/RESTART/DELETE CONTAINERS AND DESPITE BEING VERY UNLIKELY THERE IS NO CHECK OR CONCERN ABOUT ANY REMOTE CODE EXECUTION
