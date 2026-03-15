<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test xsl:fallback inside xsl:namespace. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
      <out>
         <inner a1="test">
            <t:namespace name="foo">
               <t:fallback>          
                  <t:attribute name="foo:junk" namespace="http://foo.example.com">placeholder</t:attribute>
               </t:fallback>
               <t:text>http://foo.example.com</t:text>
            </t:namespace>
         </inner>
      </out>
   </t:template>
</t:transform>
