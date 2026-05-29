# Documento de Arquitectura: Sistema de Autenticación Passwordless mediante Criptografía Asimétrica Móvil

## 1. Descripción del Proyecto
El proyecto consiste en el desarrollo de un sistema de autenticación moderno que elimina por completo el uso de contraseñas tradicionales. Su objetivo es convertir el dispositivo móvil del usuario en una **llave de seguridad de hardware** (basada en los estándares FIDO2/WebAuthn), garantizando un acceso resistente a ataques de *phishing* e ingeniería social.

El sistema utiliza **criptografía asimétrica**: el celular genera un par de llaves (Pública y Privada). La identidad del usuario se valida localmente mediante sensores biométricos (huella/rostro), lo que autoriza al chip seguro del celular a firmar digitalmente "desafíos" enviados por el servidor. Ningún secreto compartido (contraseñas, OTPs) ni dato biométrico viaja por internet.

---

## 2. Arquitectura del Sistema (El Ecosistema FIDO2)

El proyecto se divide en tres componentes principales que interactúan entre sí:

### A. La App Autenticadora (El Cliente Móvil)
Es el núcleo de seguridad del lado del usuario. Actúa como la "bóveda" física.
* **Responsabilidad:** Interactuar con el Enclave Seguro (*StrongBox/Keystore*) del celular para generar y custodiar la Llave Privada.
* **Acción:** Recibe desafíos del servidor, invoca la biometría del usuario y genera las Firmas Digitales necesarias para autorizar los inicios de sesión.

### B. El Backend (El Servidor / *Relying Party*)
Es el cerebro central que verifica la identidad criptográfica.
* **Responsabilidad:** Generar desafíos aleatorios (*challenges*), almacenar las Llaves Públicas de los usuarios registrados y verificar matemáticamente las Firmas Digitales entrantes.
* **Acción:** Otorga o deniega el acceso emitiendo tokens de sesión válidos si la verificación criptográfica es exitosa.

### C. La Aplicación Cliente (Entorno de Prueba)
Es la plataforma a la que el usuario desea acceder (para este proyecto, será una interfaz web sencilla).
* **Responsabilidad:** Actuar como puente entre el Backend y la App Autenticadora, mostrando la interfaz visual ("Iniciar Sesión") al usuario final.

---

## 3. Stack Tecnológico Seleccionado

Tras evaluar criterios de seguridad, rendimiento a nivel criptográfico y control de hardware, se ha definido el siguiente stack:

* **App Autenticadora: Kotlin (Android Nativo)**
    * *Justificación:* Permite un acceso de bajo nivel y sin intermediarios al `AndroidKeyStore` y a la API oficial `BiometricPrompt`. Esto asegura que las llaves se generen directamente en la zona segura del hardware del dispositivo, evitando vulnerabilidades de librerías multiplataforma de terceros.
* **Backend Servidor: Go (Golang)**
    * *Justificación:* Go ofrece un rendimiento excepcional en operaciones matemáticas complejas (como la verificación de firmas ECDSA) gracias a su biblioteca estándar `crypto`. Además, maneja miles de peticiones simultáneas con muy bajo consumo de recursos (mediante *Goroutines*) y posee ecosistemas maduros para implementar validaciones WebAuthn.
* **Base de Datos: PostgreSQL**
    * *Justificación:* Sistema relacional robusto y altamente eficiente para el almacenamiento de datos binarios y cadenas largas (como las Llaves Públicas en formato Base64 o blobs), garantizando la integridad de las credenciales de los usuarios.

---

## 4. Flujos Criptográficos Principales (Alto Nivel)

### 1. Fase de Registro (Creación de credenciales)
1. El usuario solicita registrarse desde el Cliente.
2. El Backend responde con un Desafío aleatorio.f
3. La App Autenticadora pide la biometría del usuario.
4. Tras el éxito biométrico, la App genera una **Llave Privada** (que se guarda en el chip del celular) y una **Llave Pública**.
5. La App empaqueta la Llave Pública y la envía al Backend.
6. El Backend guarda la Llave Pública en PostgreSQL asociada a ese usuario.

### 2. Fase de Inicio de Sesión (Validación)
1. El usuario introduce su identificador para iniciar sesión.
2. El Backend genera un Desafío **nuevo** y lo envía.
3. La App Autenticadora pide la biometría del usuario.
4. Tras el éxito biométrico, la App usa su Llave Privada almacenada para crear una **Firma Digital** sobre ese Desafío.
5. La App envía *únicamente* la Firma Digital al Backend.
6. El Backend usa la Llave Pública de su base de datos para verificar la Firma Digital. Si coincide, se concede el acceso.

---

## 5. Consideraciones de Seguridad
* **Cero Secretos Compartidos:** Si la base de datos es vulnerada, los atacantes solo obtienen Llaves Públicas, las cuales son inútiles para falsificar inicios de sesión.
* **Privacidad Biométrica:** La huella o el rostro nunca se envían al servidor ni salen del celular. Solo actúan como un interruptor físico para permitir el uso de la Llave Privada local.
* **Resistencia al Phishing:** Las llaves están vinculadas criptográficamente al dominio del servidor, impidiendo que plataformas falsas (phishing) puedan capturar firmas válidas.