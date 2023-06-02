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


function startRegister() {
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
      credentialCreationOptions.publicKey.challenge = base64url.decode(credentialCreationOptions.publicKey.challenge);
      credentialCreationOptions.publicKey.user.id = base64url.decode(credentialCreationOptions.publicKey.user.id);
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
            rawId: base64url.encode(rawId),
            type: credential.type,
            response: {
              attestationObject: base64url.encode(attestationObject),
              clientDataJSON: base64url.encode(clientDataJSON),
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
