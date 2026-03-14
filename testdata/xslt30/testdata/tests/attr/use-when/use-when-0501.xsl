<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses use-when on xsl:sort to avoid ascending sort.  -->

   <t:template match="doc">
      <out>
         <t:text>
</t:text>
         <ascending>
            <t:text>
</t:text>
            <t:for-each select="item">
               <t:sort use-when="false()"/>
               <t:copy-of select="."/>
               <t:text>
</t:text>
            </t:for-each>
         </ascending>
         <t:text>
</t:text>
         <descending>
            <t:text>
</t:text>
            <t:for-each select="item">
               <t:sort order="descending"
                       use-when="true()"
                       collation="http://www.w3.org/2005/xpath-functions/collation/codepoint"/>
               <t:copy-of select="."/>
               <t:text>
</t:text>
            </t:for-each>
         </descending>
         <t:text>
</t:text>
      </out>
  </t:template>
</t:transform>
