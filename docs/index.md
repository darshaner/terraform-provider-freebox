# Freebox Provider

The Freebox provider allows you to manage DHCP static leases and DHCP server configuration.

## Getting an `app_token` (one‑time)

Use the helper script found in the Github repo:

```bash
python3 tools/freebox_api_token.py
```

Approve on your Freebox screen when prompted, then copy the printed **APP TOKEN**.

> Keep the token secret (store in a secret manager or environment variable).

### Configuring permissions

At the time of this writing, the managment of permissions can not be done via the API. It must be done manually through the freebox OS web UI.

If you need to change the default set of permissions, first head to [http://mafreebox.freebox.fr](http://mafreebox.freebox.fr) and log in.

Then open the `Paramètres de la Freebox` menu, double click on `Gestion des accès` and switch to the `Applications` tab.

You should see the application you just registered earlier ; click on the `Editer` icon `🖉`.

Finally, pick the permissions your application requires. For a basic usage the following ones are good enough:

- `Accès au gestionnaire de téléchargements`
- `Accès aux fichiers de la Freebox`
- `Modification des réglages de la Freebox`
- `Contrôle de la VM`


## Example Usage

```hcl
provider "freebox" {
  app_token = var.freebox_app_token
  # base_url = "http://mafreebox.freebox.fr/api/v8" # optional
}
````

## Schema

### Required

* **app\_token** (String) Freebox application token (after approving the app on the Freebox).

### Optional

* **base\_url** (String) Freebox API base URL. Defaults to `http://mafreebox.freebox.fr/api/v8`.