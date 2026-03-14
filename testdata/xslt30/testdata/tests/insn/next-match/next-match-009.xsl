<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test next-match with default priority and AttributeTest -->

   <t:template match="@foo">
      <b xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:next-match/>
      </b>
   </t:template>

   <t:template match="@*">
      <c xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:next-match/>
      </c>
   </t:template>

   <t:template match="doc">
      <a xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:apply-templates select="@*"/>
      </a>
   </t:template>
</t:transform>
