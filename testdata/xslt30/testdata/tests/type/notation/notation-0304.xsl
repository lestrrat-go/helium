<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:one="http://notation.example.com"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="one"
             version="2.0">
<!-- Purpose: Test with xsl:for-each-group, grouping values of type xs:NOTATION.-->

   <t:template match="/one:NOTATION-enumeration">
      <out>
         <t:for-each-group select="one:NOTATION-element" group-by="@NOTATION-attribute">
  		        <t:value-of select="./@name"/>
  	      </t:for-each-group>
      </out>
   </t:template>
</t:transform>
