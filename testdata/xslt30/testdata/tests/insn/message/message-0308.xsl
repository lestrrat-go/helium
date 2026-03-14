<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.mytest.net"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Test case that uses if expression in the select value of an xsl:message 
  				instruction to decide what message should be generated.  -->

   <t:template match="docs">
   		<out>
         <t:message terminate="no"
                    select="(if (contains(terminate[1],'no')) then 18 else 0)       * (if (contains(terminate[2],'yes')) then 4 else 2)"/>
      </out>
	  </t:template>
</t:transform>
