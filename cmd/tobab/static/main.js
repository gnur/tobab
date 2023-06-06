let abortController;
let abortSignal;

function onload() {
  console.log("loaded");
  if (!window.PublicKeyCredential || !PublicKeyCredential.isConditionalMediationAvailable) {
    console.log("no window.PublicKeyCredential");
    return;
  }
  Promise.all([
    PublicKeyCredential.isConditionalMediationAvailable(),
    PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable()
  ]).then((values) => {
    if (values.every(x => x === true)) {
      let div = document.querySelector("#passkey");
      if (div) {
        div.style.display = "block";
        try {
          document.getElementById("create-account").addEventListener("submit", (a, event) => {
            console.log(a);

            a.preventDefault();
            startRegister();
          }, false);
        } catch { }
        let loginBtn = document.querySelector("#passkeyLogin")
        if (loginBtn) {
          loginBtn.addEventListener("click", e => {
            e.preventDefault();
            if (abortController) {
              abortController.abort();
            }
            startLogin()
          }, false);
          startDiscoverableLogin();
        }
      }
    }
  })
};

function showError(msg) {
  let errDiv = document.querySelector("#error-div");
  let dialog = document.querySelector("#messages");
  errDiv.innerHTML = msg;
  dialog.showModal();
}


function startRegister() {
  console.log("Register start");
  let username = document.querySelector("#username").value;
  if (username === "") {
    showError("Invalid username");
    return
  }
  fetch("/passkey/register/start",
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ "Name": username }),
    }).then(res => {
      return res.json()
    }).then(credentialCreationOptions => {
      credentialCreationOptions.publicKey.challenge = base64url.decode(credentialCreationOptions.publicKey.challenge);
      credentialCreationOptions.publicKey.user.id = base64url.decode(credentialCreationOptions.publicKey.user.id);
      if (credentialCreationOptions.publicKey.excludeCredentials) {
        for (var i = 0; i < credentialCreationOptions.publicKey.excludeCredentials.length; i++) {
          credentialCreationOptions.publicKey.excludeCredentials[i].id = base64url.decode(credentialCreationOptions.publicKey.excludeCredentials[i].id);
        }
      }

      return navigator.credentials.create({
        publicKey: credentialCreationOptions.publicKey,
      })
    }).then((credential) => {
      let attestationObject = credential.response.attestationObject;
      let clientDataJSON = credential.response.clientDataJSON;
      let rawId = credential.rawId;

      fetch("/passkey/register/finish",
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            id: credential.id,
            rawId: base64url.encode(rawId),
            type: credential.type,
            response: {
              attestationObject: base64url.encode(attestationObject),
              clientDataJSON: base64url.encode(clientDataJSON),
            },
          }),
        })
        .then(_ => {
          showError("successfully registered " + username + "!")
          document.location = "/";
          return
        })
        .catch((error) => {
          showError("failed to register " + username + "<br>" + error)
        })
    })
    .catch((error) => {
      showError("failed to register " + username + "<br>" + error)
    })
}

let startDiscoverableLogin = async () => {

  console.log("disc-in: start");

  if (window.PublicKeyCredential.isConditionalMediationAvailable) {
    console.log("Conditional UI is understood by the browser");
    if (!await window.PublicKeyCredential.isConditionalMediationAvailable()) {
      showError("Conditional UI is understood by your browser but not available");
      return;
    }
  } else {
    // Normally, this would mean Conditional Mediation is not available. However, the "current"
    // development implementation on chrome exposes availability via
    // navigator.credentials.conditionalMediationSupported. You won't have to add this code
    // by the time the feature is released.
    if (!navigator.credentials.conditionalMediationSupported) {
      showError("Your browser does not implement Conditional UI (are you running the right chrome/safari version with the right flags?)");
      return;
    } else {
      console.log("This browser understand the old version of Conditional UI feature detection");
    }
  }



  abortController = new AbortController();
  abortSignal = abortController.signal;

  let credentialRequestOptions = await fetch("/passkey/login/anystart",
    {
      method: "POST",
    }).then(res => {
      console.log("disc-in: got json");
      return res.json()
    })
  credentialRequestOptions.publicKey.challenge = base64url.decode(credentialRequestOptions.publicKey.challenge);

  console.log("disc-in: waiting for assertion");

  let assertion = await navigator.credentials.get({
    signal: abortSignal,
    mediation: "conditional",
    publicKey: credentialRequestOptions.publicKey
  })
  console.log("disc-in: got assertion");
  let authData = assertion.response.authenticatorData;
  let clientDataJSON = assertion.response.clientDataJSON;
  let rawId = assertion.rawId;
  let sig = assertion.response.signature;
  let userHandle = assertion.response.userHandle;


  fetch("/passkey/login/finish", {
    method: "POST",
    headers: { "Content-Type": "application/json", },
    body: JSON.stringify({
      id: assertion.id,
      rawId: base64url.encode(rawId),
      type: assertion.type,
      response: {
        authenticatorData: base64url.encode(authData),
        clientDataJSON: base64url.encode(clientDataJSON),
        signature: base64url.encode(sig),
        userHandle: base64url.encode(userHandle),
      },
    }),
  })
    .then(success => {
      console.log("disc-in: finished login");
      document.location = document.location;
      return
    })
    .catch((error) => {
      showError("failed to login " + username + "<br>" + error)
    })
}

function startLogin(event) {
  try {
    event.preventDefault();
  } catch { };

  abortController
  let username = document.querySelector("#username").value;
  if (username == "") {
    showError("Invalid username");
    return
  }

  fetch("/passkey/login/start",
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ "Name": username }),
    }).then(res => {
      return res.json()
    }).then(credentialRequestOptions => {
      credentialRequestOptions.publicKey.challenge = base64url.decode(credentialRequestOptions.publicKey.challenge);
      credentialRequestOptions.publicKey.allowCredentials.forEach(function (listItem) {
        listItem.id = base64url.decode(listItem.id)
      });

      return navigator.credentials.get({
        publicKey: credentialRequestOptions.publicKey
      })
    }).then((assertion) => {
      console.log(assertion)
      let authData = assertion.response.authenticatorData;
      let clientDataJSON = assertion.response.clientDataJSON;
      let rawId = assertion.rawId;
      let sig = assertion.response.signature;
      let userHandle = assertion.response.userHandle;


      fetch("/passkey/login/finish",
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },

          body: JSON.stringify({
            id: assertion.id,
            rawId: base64url.encode(rawId),
            type: assertion.type,
            response: {
              authenticatorData: base64url.encode(authData),
              clientDataJSON: base64url.encode(clientDataJSON),
              signature: base64url.encode(sig),
              userHandle: base64url.encode(userHandle),
            },
          }),
        })
        .then(success => {
          document.location = document.location;
          return
        })
        .catch((error) => {
          showError("failed to login " + username + "<br>" + error)
        })
    })
    .catch((error) => {
      showError("failed to register " + username + "<br>" + error)
    })
}


window.onload = onload();

//thest
