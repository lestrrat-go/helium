<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.mytest.net"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Test case that uses for expression in the select value of an xsl:message instruction.  -->

   <t:template match="docs">
   		<out>
         <t:message terminate="no" select="for $i in 1 to 3 return string(child::*[2])"/>
      </out>
	  </t:template>
</t:transform>
