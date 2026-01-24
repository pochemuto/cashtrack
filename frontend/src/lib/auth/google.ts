type GoogleCredentialHandler = (response: google.accounts.id.CredentialResponse) => void;

export function initializeGoogleSignIn(
    buttonEl: HTMLDivElement | null,
    clientId: string,
    onCredential: GoogleCredentialHandler
): boolean {
    const googleApi = window.google;
    if (!googleApi?.accounts?.id) {
        return false;
    }

    googleApi.accounts.id.initialize({
        client_id: clientId,
        callback: onCredential,
        auto_select: false,
    });

    if (buttonEl) {
        googleApi.accounts.id.renderButton(buttonEl, {
            type: "standard",
            theme: "outline",
            size: "large",
            text: "signin_with",
            shape: "pill",
        });
    }

    googleApi.accounts.id.prompt();
    return true;
}

export function cancelGoogleSignIn() {
    window.google?.accounts?.id?.cancel?.();
}
