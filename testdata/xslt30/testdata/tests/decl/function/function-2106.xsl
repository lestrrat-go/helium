<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://example.com/namespace"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0"
             exclude-result-prefixes="my">
<!-- Purpose: Test that an error is raised if xsl:param is not 
  				the first element within a stylesheet function.-->

   <t:output method="xml"/>

   <t:function name="my:pi">
      <t:value-of select="'tempo'"/>
      <t:param name="par"/>
   </t:function>

   <t:template match="/">
      <out>
         <t:value-of select="my:pi()"/>
      </out>
   </t:template>
</t:transform>
