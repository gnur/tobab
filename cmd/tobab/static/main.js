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
      document.querySelector("#passkey").style.display = "block";
    }
  })
};

// Base64 to ArrayBuffer
function bufferDecode(value) {
  return Uint8Array.from(atob(value), c => c.charCodeAt(0));
}

// ArrayBuffer to URLBase64
function bufferEncode(value) {
  return btoa(String.fromCharCode.apply(null, new Uint8Array(value)))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=/g, "");;
}

const { startAuthentication } = SimpleWebAuthnBrowser;
const { startRegistration } = SimpleWebAuthnBrowser;


async function startRegister() {
  const resp = await fetch("/passkey/register/start",
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ "Name": "testname4" }),
    });

  let attResp;
  try {
    // Pass the options to the authenticator and wait for a response
    attResp = await startRegistration(await resp.json());
  } catch (error) {
    // Some basic error handling
    throw error;
  }

  // POST the response to the endpoint that calls
  // @simplewebauthn/server -> verifyAuthenticationResponse()
  const verificationResp = await fetch('/passkey/register/finish', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(attResp),
  });

  // Wait for the results of verification
  const verificationJSON = await verificationResp.json();
  // Show UI appropriate for the `verified` status
  if (verificationJSON && verificationJSON.verified) {
    console.log("success!");
  } else {
    console.log(verificationJSON);
  }

}


function startRegistrationOld() {
  fetch("/passkey/register/start",
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ "Name": "testname4" }),
    }).then(res => {
      return res.json()
    }).then(credentialCreationOptions => {
      console.log(credentialCreationOptions)
      console.log(credentialCreationOptions.publicKey);
      credentialCreationOptions.publicKey.challenge = bufferDecode(credentialCreationOptions.publicKey.challenge);
      credentialCreationOptions.publicKey.user.id = bufferDecode(credentialCreationOptions.publicKey.user.id);
      if (credentialCreationOptions.publicKey.excludeCredentials) {
        for (var i = 0; i < credentialCreationOptions.publicKey.excludeCredentials.length; i++) {
          credentialCreationOptions.publicKey.excludeCredentials[i].id = bufferDecode(credentialCreationOptions.publicKey.excludeCredentials[i].id);
        }
      }

      return navigator.credentials.create({
        publicKey: credentialCreationOptions.publicKey
      })
    }).then((credential) => {
      console.log(credential)
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
            rawId: bufferEncode(rawId),
            type: credential.type,
            response: {
              attestationObject: bufferEncode(attestationObject),
              clientDataJSON: bufferEncode(clientDataJSON),
            },
          }),
        })
        .then(_ => {
          alert("successfully registered " + username + "!")
          return
        })
        .catch((error) => {
          console.log(error)
          alert("failed to register " + username)
        })
    })
}
window.onload = onload();

//thest
