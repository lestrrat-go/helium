<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:one="http://notation.example.com"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="one"
             version="2.0">
<!-- Purpose: Test with xsl:key and key() selecting nodes with attribute values of type xs:NOTATION.-->

   <t:key name="mykey" match="one:NOTATION-element" use="@NOTATION-attribute"/>

   <t:variable name="var"
               select="data(/one:NOTATION-enumeration/one:NOTATION-element[1]/@NOTATION-attribute)"/>

   <t:template match="/one:NOTATION-enumeration">
      <out>
         <t:value-of select="key('mykey', $var)/@name"/>
      </out>
   </t:template>
</t:transform>
