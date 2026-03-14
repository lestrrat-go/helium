<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of single AVT with value 'no' in @terminate of xsl:message. 
  				AVT value comes from a node or a variable.-->

   <t:output method="xml" encoding="UTF-8"/>

   <t:variable name="var" select="/doc/str"/>

   <t:template match="/doc">
      <out>
         <t:message select="string(msg)" terminate="{str}"/>
         <t:message select="string(msg)" terminate="{$var}"/>
         <t:value-of select="'Printing.'"/>
      </out>
   </t:template>
</t:transform>
