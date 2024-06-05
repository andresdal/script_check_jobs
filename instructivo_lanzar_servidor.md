### Paso 1: Compilar tu programa Go

Primero, compila tu programa Go para crear un ejecutable.
```
go build -o fetch_jobs_service go_to_jobs.go
```

Esto generará un archivo ejecutable llamado `mi_servicio`.

### Paso 2: Crear un archivo de servicio `systemd`
Crea un archivo de servicio `systemd` para tu programa. Este archivo debe estar en el directorio `/etc/systemd/system/`. Supongamos que llamamos a nuestro servicio `mi_servicio.service`.
```
sudo nano /etc/systemd/system/fetch_jobs_service.service
```

```init
[Unit]
Description=Mi Servicio Go
After=network.target

[Service]
ExecStart=/home/script_check_jobs/fetch_jobs_service
Restart=always
User=nobody
Group=nogroup

[Install]
WantedBy=multi-user.target
```

Asegúrate de reemplazar `/path/to/mi_servicio` con la ruta real donde se encuentra tu ejecutable. También puedes ajustar el `User` y `Group` según sea necesario.
### Paso 3: Recargar `systemd` y habilitar tu servicio
Después de crear el archivo de servicio, recarga la configuración de `systemd` para que reconozca el nuevo servicio:
```
sudo systemctl daemon-reload
```

Luego, habilita tu servicio para que se inicie automáticamente al arrancar el sistema:
```
sudo systemctl enable fetch_jobs_service
```
### Paso 4: Iniciar tu servicio

Inicia tu servicio manualmente por primera vez:
```
sudo systemctl start fetch_jobs_service
```

### Paso 5: Verificar el estado del servicio
Puedes verificar el estado de tu servicio para asegurarte de que se está ejecutando correctamente:
```
sudo systemctl status fetch_jobs_service
```

Este comando te mostrará el estado actual de tu servicio, incluyendo cualquier error que pueda haber ocurrido.

### Para detener el servicio
```
sudo systemctl stop fetch_jobs_service
```

### Modificar script
Una vez realizados los cambios, recompila tu script para generar el nuevo ejecutable.
```
go build -o fetch_jobs_service go_to_jobs.go
```

Después de actualizar el ejecutable, reinicia el servicio para que los cambios surtan efecto.
```
sudo systemctl restart fetch_jobs_service
```
