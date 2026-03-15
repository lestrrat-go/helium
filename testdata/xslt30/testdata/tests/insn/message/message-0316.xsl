<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.mytest.net"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Test terminate value other than yes | no.  -->

   <t:template match="docs">
   		<out>
         <t:message select="'Error message'" terminate="NO"/>
      </out>
	  </t:template>
</t:transform>
