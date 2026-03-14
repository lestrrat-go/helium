<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses Quantified expression (every) in the select value of an xsl:message instruction.  -->

   <t:template match="/">
   		<out>
         <t:message terminate="no"
                    select="concat('Error message: ', if ( every $v in docs/terminate satisfies string($v)='no'  )      then 'terminate=yes'      else 'terminate=no')"/>
      </out>
	  </t:template>
</t:transform>
