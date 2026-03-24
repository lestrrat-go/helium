<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test for nested xsl:result-document instruction using the principal output URI -->

   <t:output method="xhtml"/>

   <t:template match="/">
      <out>
         <t:result-document href="one.xml">
            <one>
               <t:result-document>
                  <two/>
               </t:result-document>
            </one>
         </t:result-document>
      </out>
   </t:template>
</t:transform>
