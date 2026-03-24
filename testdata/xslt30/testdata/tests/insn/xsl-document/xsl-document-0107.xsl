<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test case that uses xsl:for-each inside an xsl:document instruction to 
  				create children for the newly created document. -->

   <t:template match="root">
      <t:document>
         <out>
            <t:for-each select="child">
               <a>
                  <t:value-of select="."/>
               </a>
            </t:for-each>
         </out>
      </t:document>
   </t:template>
</t:transform>
