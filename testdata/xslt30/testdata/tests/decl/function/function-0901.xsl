<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<x:stylesheet
                     version = "2.0"
                     xmlns:x = "http://www.w3.org/1999/XSL/Transform"
                     xmlns:f = "http://local/functions"
                    xmlns:xs = "http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes = "f xs"
>

<!-- A user-supplied test that once revealed a bug -->

  <x:template match="/">
    <testing>
      <x:value-of select="f:test()" separator=","/>
    </testing>
  </x:template>

  <x:function name="f:test" as="xs:boolean*">
    <x:sequence select="for $a in 1 to 2 return f:test2()" />
  </x:function>

  <x:function name="f:test2" as="xs:boolean">
    <x:sequence select="true()" />
  </x:function>
  
</x:stylesheet> 
