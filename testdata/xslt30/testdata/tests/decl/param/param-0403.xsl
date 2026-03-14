<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Tests parameter specified within xsl:apply-imports instruction
       and a global parameter with an identical name as the template parameter, 
       which should be ignored. -->

   <t:import href="import2.xsl"/>

   <t:param name="Ppar1" select="'globalPar'"/>

   <t:variable name="vpar1" select="'globalVar'"/>

   <t:template match="doc">
      <out>
         <t:apply-imports>
            <t:with-param name="Ppar1" select="'Assigned Value'"/>
            <t:with-param name="vpar1" select="'Assigned Value'"/>
         </t:apply-imports>
      </out>
   </t:template>
</t:transform>
