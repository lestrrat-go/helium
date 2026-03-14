<?xml version="1.0"?>

<!-- Saxon 7.4 bug  -->

<x:stylesheet version="2.0" xmlns:x="http://www.w3.org/1999/XSL/Transform"
  xmlns:fn="http://local/functions" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="fn xs">

  <x:key name="key1" match="*" use="@name"/>

  <x:template match="/">
    <testing>
      <x:if test="fn:test(.)">
        <x:text>true</x:text>
      </x:if>
    </testing>
  </x:template>

  <x:function name="fn:test" as="xs:boolean">
    <x:param as="document-node()" name="e"/>
    <x:sequence select="exists($e/key('key1','s'))"/>
  </x:function>

</x:stylesheet>
