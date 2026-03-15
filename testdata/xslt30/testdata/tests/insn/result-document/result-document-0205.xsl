<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose:  Test that the primary result document can be written after a secondary 
  				 one by use of nested xsl:result-document with no arguments. -->

   <t:output method="xml" indent="no"/>

   <t:template match="/">
      <t:result-document href="multresult7.out">
         <doc-secondary>
            <out>To secondary tree</out>
            <t:result-document>
               <doc-primary>To primary tree.</doc-primary>
            </t:result-document>
            <out>Also to secondary Tree</out>
         </doc-secondary>
      </t:result-document>
   </t:template>
</t:transform>
