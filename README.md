El script go_to_jobs.go llama a los workers, cada uno con un job. Los jobs estan en la carpeta "jobs".
Descripciones de los jobs:
- fetchJobs: verifica que los jobs de las APIs de Hirable y Walla funcionen (codigo 200).
- checkJobCounts: verifica que los contadores de los jobs de todos los sitios de la API jobCount no quede "estancados" en un mismo valor, sino que vayan modificandose.
- checkEndpoints: verifica que los sitios de la API de jobs de Hireable, que tiene un endpoint con una lista de sitios, devuelvan un codigo 200.
