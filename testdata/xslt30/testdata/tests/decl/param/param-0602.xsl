<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Tests parameter specified within xsl:apply-import instruction 
  			     and a global parameter with an identical name as the template 
  			     parameter, which should be ignored. -->

   <t:import href="var_import.xsl"/>

   <t:param name="par1" select="'global'"/>

   <t:template match="doc">
      <out>
         <t:apply-imports>
            <t:with-param name="par1" select="'abc'"/>
         </t:apply-imports>
      </out>
   </t:template>

   <t:template match="text()"/>
</t:transform>
