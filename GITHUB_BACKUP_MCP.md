# Conexión MCP con GitHub para Respaldo de Código

Este documento describe la configuración y el flujo de trabajo para que cualquier agente de IA (Antigravity, Claude, Cursor, Cline, etc.) pueda conectarse mediante el **Model Context Protocol (MCP)** a tu cuenta de GitHub y gestionar respaldos automáticos del proyecto.

---

## 1. Archivos de Configuración MCP Creados

- **[.agents/mcp_config.json](file:///c:/Users/admin/Desktop/sofware%20para%20restaurantes/.agents/mcp_config.json)**: Configuración nativa del workspace para Antigravity.
- **[mcp_config.json](file:///c:/Users/admin/Desktop/sofware%20para%20restaurantes/mcp_config.json)**: Configuración en raíz para clientes MCP estándar.

Servidor MCP oficial configurado:
`@modelcontextprotocol/server-github` ejecutado a través de `npx`.

---

## 2. Requisito: Personal Access Token (PAT) de GitHub

Para que el servidor MCP pueda conectarse a tu cuenta de GitHub:

1. Ingresa a GitHub en [github.com/settings/tokens](https://github.com/settings/tokens).
2. Genera un **Personal Access Token (classic)** o un **Fine-grained token**.
3. Asegúrate de marcar los permisos mínimos necesarios:
   - `repo` (Acceso completo a repositorios: crear, leer, escribir, hacer push).
4. Copia tu token (comienza por `ghp_...`).
5. Coloca el token en el campo `"GITHUB_PERSONAL_ACCESS_TOKEN"` dentro de `mcp_config.json` o configúralo como variable de entorno del sistema:
   ```json
   "env": {
     "GITHUB_PERSONAL_ACCESS_TOKEN": "ghp_TU_TOKEN_AQUI"
   }
   ```
   *(Nota: `.gitignore` ya está configurado para evitar subir claves privadas si usas variables de entorno en `.env`)*.

---

## 3. Capacidades Disponibles para el Agente vía MCP

Una vez iniciada la sesión MCP, cualquier agente tendrá acceso a las siguientes herramientas para realizar el respaldo:

- `create_repository`: Crea el repositorio remoto si aún no existe.
- `push_files`: Envía los archivos y directorios del proyecto en un solo commit directamente a GitHub.
- `create_or_update_file`: Crea o actualiza archivos individuales en una rama de respaldo.
- `search_repositories` y `get_file_contents`: Inspecciona el estado del repositorio remoto.

---

## 4. Instrucción Directa para el Agente que realice el Respaldo

Cuando desees que el agente respalde el proyecto, simplemente pídele:

> *"Usa la conexión MCP de GitHub para respaldar este proyecto en mi repositorio [nombre-del-repo]. Asegúrate de excluir archivos temporales según el .gitignore."*

El agente ejecutará las herramientas de `@modelcontextprotocol/server-github` automáticamente.
