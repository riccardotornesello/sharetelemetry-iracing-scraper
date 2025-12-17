import os
import json
from google.cloud import pubsub_v1
from concurrent.futures import TimeoutError

from conf import PROJECT_ID


# Setup the environment variables
os.environ["PUBSUB_EMULATOR_HOST"] = "localhost:8085"

# Configuration
SUBSCRIPTION_ID = "sub-api-res" 


def callback(message):
    print(f"\n📩 Message received! (ID: {message.message_id})")
    
    # Print message attributes
    if message.attributes:
        print("   🏷️  Attributes:", dict(message.attributes))

    # 2. Decoding the payload
    payload_bytes = message.data
    try:
        # Attempt to decode JSON for "clean" printing
        payload_str = payload_bytes.decode("utf-8")
        payload_json = json.loads(payload_str)
        print("   📦 Payload (JSON):")
        print(json.dumps(payload_json, indent=4)) # Pretty print
    except (UnicodeDecodeError, json.JSONDecodeError):
        # If it's not JSON or not text, print the raw representation
        print(f"   📦 Payload (Raw): {payload_bytes}")

    # 3. ACK (Acknowledge)
    # IMPORTANT: Tells Pub/Sub "I'm done, delete the message from the queue".
    # If you don't do this, Pub/Sub will resend it indefinitely after the ack deadline.
    message.ack()

def start_listening():
    subscriber = pubsub_v1.SubscriberClient()
    subscription_path = subscriber.subscription_path(PROJECT_ID, SUBSCRIPTION_ID)

    print(f"🎧 Listening on subscription: {subscription_path}")
    print("Press CTRL+C to exit.\n")

    # Pass the 'callback' function to the client
    streaming_pull_future = subscriber.subscribe(subscription_path, callback=callback)

    # The 'with' block automatically manages the client's closure
    with subscriber:
        try:
            # result() blocks the script indefinitely waiting for messages
            streaming_pull_future.result()
        except TimeoutError:
            streaming_pull_future.cancel()
        except KeyboardInterrupt:
            # Clean exit on CTRL+C
            streaming_pull_future.cancel()
            print("\n🛑 Stop listening.")

if __name__ == "__main__":
    start_listening()