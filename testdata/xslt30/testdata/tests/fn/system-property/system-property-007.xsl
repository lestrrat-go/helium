<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:t="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test that xsl:supports-serialization is a valid system-property and the returned result is an xs:string.
  				Conditional xsl:if used to verify result value is 'yes' or 'no'-->

   <t:template match="/">
      <out>
         <t:value-of select="system-property('t:supports-serialization') instance of xs:string"/>
         <t:if test="system-property('t:supports-serialization')='yes' or system-property('t:supports-serialization')='no'">
    	       <t:value-of select="'true'"/>
         </t:if>
      </out>
   </t:template>
</t:transform>
