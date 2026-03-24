<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Simple test of next-match where the match pattern is a NameTest -->

   <t:template match="doc">
      <a xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:next-match/>
      </a>
   </t:template>

   <t:template match="foo">
      <c xmlns:xsl="http://www.w3.org/1999/XSL/Transform"/>
   </t:template>

   <t:template match="*">
      <b xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:apply-templates/>
      </b>
   </t:template>
   
   <t:template match="*[some $x in * satisfies name($x) = 'foobar']">
      <b xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:apply-templates/>
      </b>
   </t:template>
</t:transform>
