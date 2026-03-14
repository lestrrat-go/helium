<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose:  Test AVT in the @href of an xsl:result-document.  -->

   <t:output name="nine999" method="html"/>

   <t:output method="xml" indent="no"/>

   <t:param name="filename">mult/multresult19.out</t:param>

   <t:template match="/">
      <t:result-document href="">
         <out>This should be in the primary xml document</out>
      </t:result-document>
      <t:result-document href="{$filename}" format="nine999">
         <t:text>This should be in the secondary text document</t:text>
      </t:result-document>
   </t:template>
</t:transform>
