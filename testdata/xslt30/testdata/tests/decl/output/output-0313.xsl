<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <t:import href="output-0313a.xsl"/>

   <t:output name="n"
             method="xml"
             doctype-system=""
             doctype-public=""
             indent="no"
             omit-xml-declaration="yes"/>

   <t:param name="zls" select="''"/>

   <t:template match="/">
      <t:result-document format="n" doctype-system="{$zls}" doctype-public="{$zls}">
         <a>
            <b/>
         </a>
      </t:result-document>
   </t:template>
</t:transform>
