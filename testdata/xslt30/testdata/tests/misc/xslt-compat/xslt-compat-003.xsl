<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test xsl:fallback inside xsl:result-document. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="DATA">
      <out>
         <log>((</log>
         <t:result-document href="xslt-compat-003-2.out">
            <t:fallback>          
                  <t:attribute name="foo:junk" namespace="http://foo.example.com">placeholder</t:attribute>
            </t:fallback>
            <t:copy-of select="item[2]"/>
         </t:result-document>
         <log>))</log>
      </out>
   </t:template>
</t:transform>
