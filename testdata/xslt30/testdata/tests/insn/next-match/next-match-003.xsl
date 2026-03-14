<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test next-match inside templates with pattern of NameTest, AnyKindTest (node()) or '*' 
  				with predicate. All have computed priorities. -->

   <t:template match="node()">
      <a xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:next-match/>
      </a>
   </t:template>

   <t:template match="doc">
      <b xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:next-match/>
      </b>
   </t:template>

   <t:template match="*[foo]">
      <c xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:next-match/>
      </c>
   </t:template>
</t:transform>
