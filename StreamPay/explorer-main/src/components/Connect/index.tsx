import { useEffect, useState } from 'react'
import {
  Stack,
  FormControl,
  Button,
  useColorModeValue,
  Heading,
  Text,
  Container,
  Flex,
} from '@chakra-ui/react'
import { CheckIcon } from '@chakra-ui/icons'
import { useDispatch } from 'react-redux'
import {
  setConnectState,
  setTmClient,
  setRPCAddress,
} from '@/store/connectSlice'
import Head from 'next/head'
import { LS_RPC_ADDRESS, LS_RPC_ADDRESS_LIST } from '@/utils/constant'
import { validateConnection, connectWebsocketClient } from '@/rpc/client'

export default function Connect() {
  const [state, setState] = useState<'initial' | 'submitting' | 'success'>(
    'initial'
  )
  const [error, setError] = useState<string | null>(null)
  const dispatch = useDispatch()

  const rpcAddress =
    process.env.NEXT_PUBLIC_RPC_ADDRESS || 'http://127.0.0.1:26657'

  useEffect(() => {
    if (rpcAddress) {
      connectClient(rpcAddress)
    } else {
      console.error('Environment variable NEXT_PUBLIC_RPC_ADDRESS is missing.')
      setError('Environment variable NEXT_PUBLIC_RPC_ADDRESS is missing.')
    }
  }, [rpcAddress])

  const connectClient = async (rpcAddress: string) => {
    try {
      setError(null)
      setState('submitting')

      const isValid = await validateConnection(rpcAddress)
      if (!isValid) {
        throw new Error(`Can not connect to address: ${rpcAddress}`)
      }

      const tmClient = await connectWebsocketClient(rpcAddress)
      if (!tmClient) {
        throw new Error(
          `Failed to connect to the websocket client at ${rpcAddress}`
        )
      }

      dispatch(setConnectState(true))
      dispatch(setTmClient(tmClient))
      dispatch(setRPCAddress(rpcAddress))
      setState('success')

      window.localStorage.setItem(LS_RPC_ADDRESS, rpcAddress)
      window.localStorage.setItem(
        LS_RPC_ADDRESS_LIST,
        JSON.stringify([rpcAddress])
      )
    } catch (err) {
      const errorMessage =
        err instanceof Error ? err.message : 'An unknown error occurred.'
      console.error(errorMessage)
      setError(errorMessage)
      setState('initial')
    }
  }

  return (
    <>
      <Head>
        <title>Blockchain Explorer | Connect</title>
        <meta
          name="description"
          content="Blockchain Explorer | Connect to RPC Address"
        />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <link rel="icon" href="/favicon.ico" />
      </Head>
      <Flex
        minH={'100vh'}
        align={'center'}
        justify={'center'}
        bg={useColorModeValue('light-bg', 'dark-bg')}
        flexDirection={'column'}
        gap={16}
      >
        <Container
          maxW={'lg'}
          bg={useColorModeValue('light-container', 'dark-container')}
          boxShadow={'xl'}
          rounded={'lg'}
          p={6}
        >
          <Heading
            as={'h2'}
            fontSize={{ base: '2xl', sm: '3xl' }}
            textAlign={'center'}
            fontFamily="monospace"
            fontWeight="bold"
          >
            SoCone Blockchain Explorer
          </Heading>
          <Text as={'h2'} fontSize="lg" textAlign={'center'} mb={5}>
            Connecting to RPC Address... {rpcAddress}
          </Text>
          <Stack direction={'column'} spacing={4} align="center">
            <FormControl>
              <Button
                backgroundColor={useColorModeValue('light-theme', 'dark-theme')}
                color={'white'}
                _hover={{
                  backgroundColor: useColorModeValue(
                    'dark-theme',
                    'light-theme'
                  ),
                }}
                isLoading={state === 'submitting'}
                disabled={state === 'success'}
                type="button"
              >
                {state === 'success' ? <CheckIcon /> : 'Connecting...'}
              </Button>
            </FormControl>
            <Text
              mt={2}
              textAlign={'center'}
              color={error ? 'red.500' : 'gray.500'}
            >
              {error ? `Error: ${error}` : ''}
            </Text>
          </Stack>
        </Container>
      </Flex>
    </>
  )
}
